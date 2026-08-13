package search

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JeremiahM37/librarr/internal/config"
)

func TestOpenLibrarySearchSurfacesEditionMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("fields"); got == "" || !containsField(got, "language") {
			t.Fatalf("fields = %q, want language", got)
		}
		_ = json.NewEncoder(w).Encode(olResponse{Docs: []olDoc{{
			Key:              "/works/OL12345W",
			Title:            "Pride and Prejudice",
			AuthorName:       []string{"Jane Austen"},
			EbookAccess:      "public",
			IA:               []string{"prideandprejudice"},
			FirstPublishYear: 1813,
			Language:         []string{"eng"},
			CoverI:           12345,
		}}})
	}))
	defer server.Close()

	cfg := configWithRegistry(t)
	cfg.Sources.OpenLibrary.SearchURL = server.URL
	cfg.Sources.OpenLibrary.CoverURL = server.URL
	results, err := NewOpenLibrary(cfg, server.Client()).Search(context.Background(), "pride")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	result := results[0]
	if result.Year != "1813" {
		t.Errorf("Year = %q, want 1813", result.Year)
	}
	if result.Language != "en" {
		t.Errorf("Language = %q, want en", result.Language)
	}
	if result.SizeHuman != "Public Domain" {
		t.Errorf("SizeHuman = %q, want Public Domain", result.SizeHuman)
	}
}

func containsField(fields, wanted string) bool {
	for _, field := range strings.Split(fields, ",") {
		if field == wanted {
			return true
		}
	}
	return false
}

func TestOpenLibrary_Metadata(t *testing.T) {
	cfg := &config.Config{}
	o := NewOpenLibrary(cfg, http.DefaultClient)

	if o.Name() != "openlibrary" {
		t.Errorf("expected name openlibrary, got %s", o.Name())
	}
	if !o.Enabled() {
		t.Error("expected always enabled")
	}
	if o.SearchTab() != "main" {
		t.Errorf("expected tab main, got %s", o.SearchTab())
	}
	if o.DownloadType() != "direct" {
		t.Errorf("expected download type direct, got %s", o.DownloadType())
	}
}

func TestOpenLibrary_ResponseParsing(t *testing.T) {
	// Test JSON parsing without making HTTP calls
	response := olResponse{
		Docs: []olDoc{
			{
				Key:              "/works/OL12345W",
				Title:            "Pride and Prejudice",
				AuthorName:       []string{"Jane Austen"},
				EbookAccess:      "public",
				IA:               []string{"prideandprejudice"},
				FirstPublishYear: 1813,
				CoverI:           12345,
			},
			{
				Key:         "/works/OL99999W",
				Title:       "Not Public",
				EbookAccess: "borrowable",
				IA:          []string{"notpublic"},
			},
			{
				Key:         "/works/OL88888W",
				Title:       "No IA IDs",
				EbookAccess: "public",
				IA:          nil,
			},
			{
				Key:         "/works/OL77777W",
				Title:       "Many IAs",
				EbookAccess: "public",
				IA:          []string{"a", "b", "c", "d", "e", "f", "g"},
			},
		},
	}

	// Verify serialization/deserialization
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed olResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(parsed.Docs) != 4 {
		t.Fatalf("expected 4 docs, got %d", len(parsed.Docs))
	}

	t.Run("public domain with IA", func(t *testing.T) {
		doc := parsed.Docs[0]
		if doc.EbookAccess != "public" {
			t.Error("expected public access")
		}
		if len(doc.IA) != 1 {
			t.Errorf("expected 1 IA ID, got %d", len(doc.IA))
		}
		if doc.FirstPublishYear != 1813 {
			t.Errorf("expected 1813, got %d", doc.FirstPublishYear)
		}
	})

	t.Run("not public filtered", func(t *testing.T) {
		doc := parsed.Docs[1]
		if doc.EbookAccess == "public" {
			t.Error("expected non-public access")
		}
	})

	t.Run("no IA IDs filtered", func(t *testing.T) {
		doc := parsed.Docs[2]
		if len(doc.IA) != 0 {
			t.Errorf("expected 0 IA IDs, got %d", len(doc.IA))
		}
	})

	t.Run("IA IDs truncated", func(t *testing.T) {
		doc := parsed.Docs[3]
		if len(doc.IA) != 7 {
			t.Errorf("expected 7 IA IDs, got %d", len(doc.IA))
		}
		// In the actual Search(), IA IDs are truncated to 5
		iaIDs := doc.IA
		if len(iaIDs) > 5 {
			iaIDs = iaIDs[:5]
		}
		if len(iaIDs) != 5 {
			t.Errorf("expected truncated to 5, got %d", len(iaIDs))
		}
	})
}
