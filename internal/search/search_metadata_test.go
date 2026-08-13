package search

import "testing"

func TestNormalizeSearchLanguage(t *testing.T) {
	tests := map[string]string{
		"English":          "en",
		"RUS":              "ru",
		"pt":               "pt",
		"English, Russian": "en",
		"Klingon":          "klingon",
		"":                 "",
	}
	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			if got := normalizeSearchLanguage(input); got != want {
				t.Errorf("normalizeSearchLanguage(%q) = %q, want %q", input, got, want)
			}
		})
	}
}

func TestNormalizeSearchYear(t *testing.T) {
	tests := []struct {
		input any
		want  string
	}{
		{2014, "2014"},
		{float64(2015), "2015"},
		{"1869-01-01", "1869"},
		{"2020", "2020"},
		{"2020 edition", ""},
		{float64(2015.5), ""},
		{999, ""},
	}
	for _, test := range tests {
		if got := normalizeSearchYear(test.input); got != test.want {
			t.Errorf("normalizeSearchYear(%v) = %q, want %q", test.input, got, test.want)
		}
	}
}
