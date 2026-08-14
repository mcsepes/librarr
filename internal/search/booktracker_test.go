package search

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/JeremiahM37/librarr/internal/config"
	"github.com/PuerkitoBio/goquery"
)

// booktrackerSearchPage is a minimal phpBB-style search results page with one
// topic row. It mirrors the selectors the parser looks for: an <a.topictitle>
// whose href carries the topic ID, plus a seed cell tagged with .seedmed.
const booktrackerSearchPage = `<html><body>
<table>
 <tr>
   <td><a class="topictitle" href="viewtopic.php?t=98765">Толстой - Анна Каренина [epub, 2.3 МБ]</a></td>
   <td class="seedmed">12</td>
   <td>4</td>
 </tr>
 <tr>
   <td>header row (no topictitle, must be skipped)</td>
   <td class="seedmed">999</td>
 </tr>
 <tr>
   <td><a class="topictitle" href="viewtopic.php?t=98765">Duplicate topic</a></td>
   <td class="seedmed">7</td>
 </tr>
</table>
</body></html>`

// Helper: BookTracker server that correctly sets an authenticated cookie
// (phpbb_data-like) on login, and serves results on search.
func newBookTrackerTestServer(t *testing.T, authCookie bool) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/login.php", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("login method = %s, want POST", r.Method)
		}
		if got := r.PostFormValue("login_username"); got != "u" {
			t.Errorf("login_username = %q, want u", got)
		}
		if got := r.PostFormValue("login_password"); got == "" {
			t.Error("login_password is empty")
		}
		if got := r.PostFormValue("username"); got != "" {
			t.Errorf("legacy username field unexpectedly sent: %q", got)
		}
		if got := r.PostFormValue("password"); got != "" {
			t.Error("legacy password field unexpectedly sent")
		}

		if authCookie {
			// Mimic a phpBB persistent-login cookie (value is a serialized blob).
			http.SetCookie(w, &http.Cookie{Name: "phpbb2mysql_data", Value: "a%3A2%3A%7Bi%3A0%3Bs%3A1%3A%221%22%3B%7D", Path: "/"})
		} else {
			// Only an anonymous session cookie, not a login-bearing one.
			http.SetCookie(w, &http.Cookie{Name: "phpbb2mysql_sid", Value: "anonsessid", Path: "/"})
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/search.php", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("nm"); got == "" {
			t.Error("search request missing nm query parameter")
		}
		if got := r.URL.Query().Get("to"); got != "1" {
			t.Errorf("to = %q, want 1", got)
		}
		if got := r.URL.Query().Get("max"); got != "20" {
			t.Errorf("max = %q, want 20", got)
		}
		if got := r.URL.Query().Get("search_keywords"); got != "" {
			t.Errorf("legacy search_keywords parameter unexpectedly sent: %q", got)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(booktrackerSearchPage))
	})
	return httptest.NewServer(mux)
}

func TestBookTrackerParseSearchResultsWithoutTableRows(t *testing.T) {
	// BookTracker's current search endpoint uses div.topictitle cards instead
	// of the older table-row result markup.
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<html><body>
<div class="topictitle"><a class="topictitle" href="viewtopic.php?t=24680">Булгаков - Мастер и Маргарита [fb2, 1.2 МБ]</a></div>
</body></html>`))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	s := &BookTracker{tab: "main"}
	results := s.parseSearchResults(doc, "https://booktracker.example")
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Format != "fb2" {
		t.Errorf("Format = %q, want fb2", results[0].Format)
	}
	if results[0].URL != "https://booktracker.example/viewtopic.php?t=24680" {
		t.Errorf("URL = %q", results[0].URL)
	}
}

func TestBookTrackerSeparatesEbooksAndAudiobooks(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<html><body>
<div class="topictitle"><a class="topictitle" href="viewtopic.php?t=1">Автор - Электронная книга [epub]</a></div>
<div class="topictitle"><a class="topictitle" href="viewtopic.php?t=2">Автор - Аудиокнига [m4b]</a></div>
<div class="topictitle"><a class="topictitle" href="viewtopic.php?t=3">Автор - Без формата (аудиокнига)</a></div>
</body></html>`))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	ebooks := (&BookTracker{tab: "main"}).parseSearchResults(doc, "https://booktracker.example")
	if len(ebooks) != 1 {
		t.Fatalf("ebook results = %d, want 1", len(ebooks))
	}
	if ebooks[0].Format != "epub" || ebooks[0].MediaType != "ebook" {
		t.Errorf("ebook result = %#v", ebooks[0])
	}

	audiobooks := (&BookTracker{tab: "audiobook"}).parseSearchResults(doc, "https://booktracker.example")
	if len(audiobooks) != 2 {
		t.Fatalf("audiobook results = %d, want 2", len(audiobooks))
	}
	for _, result := range audiobooks {
		if result.MediaType != "audiobook" {
			t.Errorf("result MediaType = %q, want audiobook", result.MediaType)
		}
	}
}

func TestBookTrackerSearch(t *testing.T) {
	srv := newBookTrackerTestServer(t, true)
	defer srv.Close()

	cfg := &config.Config{
		BookTrackerURL:     srv.URL,
		BookTrackerUser:    "u",
		BookTrackerPass:    "p",
		BookTrackerEnabled: true,
		UserAgent:          "test",
	}
	s := NewBookTracker(cfg, &http.Client{Timeout: 5 * time.Second}, "main")

	if !s.Enabled() {
		t.Fatal("BookTracker should be enabled with all creds set")
	}
	if s.Name() != "booktracker" {
		t.Errorf("Name() = %q", s.Name())
	}

	results, err := s.Search(context.Background(), "анна")
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1 (duplicate topic ID must be dropped)", len(results))
	}
	r := results[0]
	if !strings.HasPrefix(r.Title, "Толстой") {
		t.Errorf("Title = %q", r.Title)
	}
	if r.Author != "Толстой" {
		t.Errorf("Author = %q, want Толстой (parsed from 'Author - Title' format)", r.Author)
	}
	if r.Format != "epub" {
		t.Errorf("Format = %q, want epub", r.Format)
	}
	if r.Seeders != 12 {
		t.Errorf("Seeders = %d, want 12 (from .seedmed cell)", r.Seeders)
	}
	if !strings.Contains(r.DownloadURL, "download.php?t=98765") {
		t.Errorf("DownloadURL = %q", r.DownloadURL)
	}
	if !strings.Contains(r.URL, "viewtopic.php?t=98765") {
		t.Errorf("URL = %q", r.URL)
	}
}

// TestBookTrackerLoginRejectsAnonymousSession verifies the login check does
// not treat a plain anonymous phpBB session cookie as success. This used to
// accept "any cookie" which silently swallowed bad-credentials logins.
func TestBookTrackerLoginRejectsAnonymousSession(t *testing.T) {
	srv := newBookTrackerTestServer(t, false)
	defer srv.Close()

	cfg := &config.Config{
		BookTrackerURL:     srv.URL,
		BookTrackerUser:    "u",
		BookTrackerPass:    "wrong",
		BookTrackerEnabled: true,
		UserAgent:          "test",
	}
	s := NewBookTracker(cfg, &http.Client{Timeout: 5 * time.Second}, "main")

	_, err := s.Search(context.Background(), "x")
	if err == nil {
		t.Fatal("expected login failure when only an anonymous session cookie is present")
	}
	if !strings.Contains(err.Error(), "login") {
		t.Errorf("err = %v, want something referencing login", err)
	}
}

func TestBookTrackerDisabled(t *testing.T) {
	cfg := &config.Config{BookTrackerEnabled: true} // missing url/user/pass
	s := NewBookTracker(cfg, &http.Client{}, "main")
	if s.Enabled() {
		t.Errorf("Enabled() true without credentials; want false")
	}
}

func TestBookTrackerAudiobookTab(t *testing.T) {
	srv := newBookTrackerTestServer(t, true)
	defer srv.Close()

	cfg := &config.Config{
		BookTrackerURL: srv.URL, BookTrackerUser: "u", BookTrackerPass: "p",
		BookTrackerEnabled: true, UserAgent: "test",
	}
	s := NewBookTracker(cfg, &http.Client{Timeout: 5 * time.Second}, "audiobook")
	if s.Name() != "booktracker_audiobook" {
		t.Errorf("Name() = %q", s.Name())
	}
	results, err := s.Search(context.Background(), "x")
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("audiobook search returned %d ebook results, want 0", len(results))
	}
	for _, r := range results {
		if r.MediaType != "audiobook" {
			t.Errorf("result MediaType = %q, want audiobook", r.MediaType)
		}
	}
}
