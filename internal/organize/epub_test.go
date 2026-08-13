package organize

import (
	"testing"
)

func TestWordOverlap(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		actual   string
		want     float64
	}{
		{"exact match", "The Great Gatsby", "The Great Gatsby", 1.0},
		{"case insensitive", "the great gatsby", "THE GREAT GATSBY", 1.0},
		{"partial match", "The Great Gatsby", "Great Stories", 0.5}, // "great" matches, "gatsby" doesn't -> 1/2 = 0.5
		{"no match", "The Great Gatsby", "Harry Potter", 0.0},
		{"empty expected", "", "Some Title", 1.0},                               // empty expected = 1.0
		{"stopwords ignored", "The Book of Everything", "Everything Goes", 0.5}, // "book" + "everything" -> only "everything" matches -> 1/2 = 0.5
		{"all stopwords", "the a an of", "different words", 1.0},                // no significant words in expected -> 1.0
		{"subtitle ignored", "Dune", "Dune: Part One", 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wordOverlap(tt.expected, tt.actual)
			if got != tt.want {
				t.Errorf("wordOverlap(%q, %q) = %f, want %f", tt.expected, tt.actual, got, tt.want)
			}
		})
	}
}

func TestExtractSignificantWords(t *testing.T) {
	tests := []struct {
		input    string
		expected map[string]bool
	}{
		{"The Great Gatsby", map[string]bool{"great": true, "gatsby": true}},
		{"a book", map[string]bool{"book": true}},
		{"", map[string]bool{}},
		{"I x y", map[string]bool{}}, // all single chars or stopwords
		{"hello-world test", map[string]bool{"hello": true, "world": true, "test": true}},
		{"Августовские пушки", map[string]bool{"августовские": true, "пушки": true}},
		{"Уловка\\_22", map[string]bool{"уловка": true, "22": true}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractSignificantWords(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("extractSignificantWords(%q) = %v, want %v", tt.input, got, tt.expected)
				return
			}
			for w := range tt.expected {
				if !got[w] {
					t.Errorf("missing word %q in result", w)
				}
			}
		})
	}
}

func TestVerifyEPUBTitle_WordOverlapLogic(t *testing.T) {
	// We can't easily test the full VerifyEPUBTitle without real EPUB files,
	// but we can test the underlying logic.

	t.Run("high overlap passes", func(t *testing.T) {
		overlap := wordOverlap("The Great Gatsby", "Great Gatsby Novel")
		if overlap < 0.8 {
			t.Errorf("expected high overlap for similar titles, got %f", overlap)
		}
	})

	t.Run("low overlap fails", func(t *testing.T) {
		overlap := wordOverlap("The Great Gatsby", "Harry Potter")
		if overlap >= 0.5 {
			t.Errorf("expected low overlap for different titles, got %f", overlap)
		}
	})

	t.Run("empty title always passes", func(t *testing.T) {
		overlap := wordOverlap("", "Any Title")
		if overlap != 1.0 {
			t.Errorf("expected 1.0 for empty expected, got %f", overlap)
		}
	})

	t.Run("Cyrillic title with source decoration passes", func(t *testing.T) {
		overlap := wordOverlap("Августовские пушки [изд. 2012]", "Августовские пушки")
		if overlap < 0.6 {
			t.Errorf("expected Russian title overlap to pass, got %f", overlap)
		}
	})

	t.Run("escaped underscore normalizes as punctuation", func(t *testing.T) {
		overlap := wordOverlap("Уловка-22", "Уловка\\_22")
		if overlap != 1.0 {
			t.Errorf("expected punctuation-only title change to match, got %f", overlap)
		}
	})
}

func TestCanonicalTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Августовские пушки [изд. 2012]", "августовские пушки"},
		{"Уловка-22", "уловка 22"},
		{"Уловка\\_22", "уловка 22"},
		{"Pattern Recognition: A Novel", "pattern recognition a novel"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := canonicalTitle(tt.input); got != tt.want {
				t.Errorf("canonicalTitle(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeAuthor(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// "Last, First" → "First Last"
		{"Wight, Will", "Will Wight"},
		{"Sanderson, Brandon", "Brandon Sanderson"},
		{"Herbert, Frank", "Frank Herbert"},
		// "Last, First Middle" → "First Middle Last"
		{"Tolkien, John Ronald Reuel", "John Ronald Reuel Tolkien"},
		// Already correct
		{"Will Wight", "Will Wight"},
		{"Brandon Sanderson", "Brandon Sanderson"},
		// Multiple authors — don't touch
		{"Author One, Author Two, Author Three", "Author One, Author Two, Author Three"},
		{"Author One & Author Two", "Author One & Author Two"},
		// Empty
		{"", ""},
		// Punctuation artifacts
		{".Will Wight.", "Will Wight"},
		{"Will Wight;", "Will Wight"},
		// Whitespace
		{"  Wight ,  Will  ", "Will Wight"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeAuthor(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeAuthor(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
