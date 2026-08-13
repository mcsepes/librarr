package search

import (
	"math"
	"strconv"
	"strings"
)

// searchLanguageCodes normalizes the language labels returned by sources into
// the compact ISO 639-1 codes displayed by the search UI. Sources that return
// an unfamiliar code or label retain that value rather than losing metadata.
var searchLanguageCodes = map[string]string{
	"arabic": "ar", "bengali": "bn", "bulgarian": "bg", "chinese": "zh",
	"czech": "cs", "danish": "da", "dutch": "nl", "english": "en",
	"finnish": "fi", "french": "fr", "german": "de", "greek": "el",
	"hebrew": "he", "hindi": "hi", "hungarian": "hu", "indonesian": "id",
	"italian": "it", "japanese": "ja", "korean": "ko", "norwegian": "no",
	"persian": "fa", "polish": "pl", "portuguese": "pt", "romanian": "ro",
	"russian": "ru", "slovak": "sk", "spanish": "es", "swedish": "sv",
	"turkish": "tr", "ukrainian": "uk", "vietnamese": "vi",
	"ara": "ar", "ben": "bn", "bul": "bg", "ces": "cs", "chi": "zh",
	"dan": "da", "deu": "de", "dut": "nl", "ell": "el", "eng": "en",
	"fin": "fi", "fra": "fr", "heb": "he", "hin": "hi", "hun": "hu",
	"ind": "id", "ita": "it", "jpn": "ja", "kor": "ko", "nld": "nl",
	"nor": "no", "per": "fa", "pol": "pl", "por": "pt", "ron": "ro",
	"rus": "ru", "slk": "sk", "spa": "es", "swe": "sv", "tur": "tr",
	"ukr": "uk", "vie": "vi", "zho": "zh",
}

func normalizeSearchLanguage(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	if code, ok := searchLanguageCodes[value]; ok {
		return code
	}
	for _, separator := range []string{",", ";", "/"} {
		if index := strings.Index(value, separator); index > 0 {
			return normalizeSearchLanguage(value[:index])
		}
	}
	return value
}

func firstSearchLanguage(values []string) string {
	for _, value := range values {
		if language := normalizeSearchLanguage(value); language != "" {
			return language
		}
	}
	return ""
}

func normalizeSearchYear(value any) string {
	var year int
	switch typed := value.(type) {
	case int:
		year = typed
	case int64:
		if typed < 1000 || typed > 2999 {
			return ""
		}
		year = int(typed)
	case float64:
		if math.Trunc(typed) != typed || typed < 1000 || typed > 2999 {
			return ""
		}
		year = int(typed)
	case string:
		typed = strings.TrimSpace(typed)
		if len(typed) >= 4 && (len(typed) == 4 || typed[4] == '-') {
			parsed, err := strconv.Atoi(typed[:4])
			if err != nil {
				return ""
			}
			year = parsed
		}
	default:
		return ""
	}
	if year < 1000 || year > 2999 {
		return ""
	}
	return strconv.Itoa(year)
}
