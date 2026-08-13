package zlibraryparse

import (
	"encoding/json"
	"fmt"
	"strings"
)

// LoginSession is the authenticated session returned by known Z-Library
// login response shapes.
type LoginSession struct {
	UserID  int
	UserKey string
}

// LoginSessionFromJSON accepts both the current eapi/user/login envelope and
// the legacy rpc.php response. Errors have changed shape over time, so this
// keeps the server's message instead of exposing a JSON type error.
func LoginSessionFromJSON(body []byte) (LoginSession, error) {
	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return LoginSession{}, err
	}
	obj, ok := root.(map[string]any)
	if !ok {
		return LoginSession{}, fmt.Errorf("zlibrary login response is not an object")
	}
	if !JSONTruthy(obj["success"], true) {
		return LoginSession{}, fmt.Errorf("zlibrary login failed: %s", ErrorMessage(obj))
	}
	if message := stringValue(obj["errors"]); message != "" {
		return LoginSession{}, fmt.Errorf("zlibrary login failed: %s", message)
	}

	session := firstObject(obj, "user", "response", "data")
	if session == nil {
		return LoginSession{}, fmt.Errorf("zlibrary login failed: missing session credentials")
	}
	result := LoginSession{
		UserID:  intValue(firstPresent(session, "id", "user_id", "userId")),
		UserKey: firstString(session, "remix_userkey", "user_key", "userKey"),
	}
	if result.UserID == 0 || result.UserKey == "" {
		message := ErrorMessage(session)
		if message == "unknown error" {
			message = ErrorMessage(obj)
		}
		if message != "unknown error" {
			return LoginSession{}, fmt.Errorf("zlibrary login failed: %s", message)
		}
		return LoginSession{}, fmt.Errorf("zlibrary login failed: missing session credentials")
	}
	return result, nil
}

// DomainsFromJSON normalizes string and object domain records returned by
// Z-Library's domain endpoint.
func DomainsFromJSON(body []byte) ([]string, error) {
	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, err
	}
	obj, ok := root.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("zlibrary domains response is not an object")
	}
	if !JSONTruthy(obj["success"], true) {
		return nil, fmt.Errorf("zlibrary domains failed: %s", ErrorMessage(obj))
	}

	items, ok := obj["domains"].([]any)
	if !ok {
		return nil, nil
	}
	domains := make([]string, 0, len(items))
	for _, item := range items {
		var domain string
		switch value := item.(type) {
		case string:
			domain = strings.TrimSpace(value)
		case map[string]any:
			domain = firstString(value, "domain", "host", "url")
		}
		if domain != "" {
			domains = append(domains, domain)
		}
	}
	return domains, nil
}

// FileDownloadFromJSON extracts an expiring URL from the current file
// endpoint, keeping an account-quota response visible to the caller.
func FileDownloadFromJSON(body []byte) (string, error) {
	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return "", err
	}
	obj, ok := root.(map[string]any)
	if !ok {
		return "", fmt.Errorf("zlibrary download response is not an object")
	}
	if !JSONTruthy(obj["success"], true) {
		return "", fmt.Errorf("zlibrary download failed: %s", ErrorMessage(obj))
	}
	file := firstObject(obj, "file", "data", "response", "result")
	if file == nil {
		return "", fmt.Errorf("zlibrary download response missing file")
	}
	if link := firstString(file, "downloadLink", "download_link", "dl", "url"); link != "" {
		return link, nil
	}
	if allowed, present := file["allowDownload"].(bool); present && !allowed {
		message := firstString(file, "disallowDownloadMessage", "message", "error")
		if message == "" {
			message = "download is not currently allowed for this account"
		}
		return "", fmt.Errorf("zlibrary download unavailable: %s", message)
	}
	return "", fmt.Errorf("zlibrary download response missing link")
}

func firstObject(obj map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		if value, ok := obj[key].(map[string]any); ok {
			return value
		}
	}
	return nil
}
