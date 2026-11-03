package main

import (
	"slices"
	"strings"
	"testing"
)

func TestEscapeLikePattern(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain text", input: "go", want: "go"},
		{name: "percent", input: "50%", want: `50\%`},
		{name: "underscore", input: "snake_case", want: `snake\_case`},
		{name: "backslash", input: `C:\temp`, want: `C:\\temp`},
		{name: "all three", input: `%_\`, want: `\%\_\\`},
		{name: "empty", input: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := escapeLikePattern(tt.input); got != tt.want {
				t.Errorf("escapeLikePattern(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSearchPages(t *testing.T) {
	conn := newTestDB(t)

	pages := []struct{ title, url, content, language string }{
		{"Go language", "https://example.com/go", "Go is a compiled language", "en"},
		{"Sales report", "https://example.com/sales", "Sales grew 50% last year", "en"},
		{"User count", "https://example.com/users", "There were 500 users", "en"},
		{"Go sprog", "https://example.com/go-da", "Go er et sprog", "da"},
	}
	for _, p := range pages {
		_, err := conn.ExecContext(t.Context(),
			`INSERT INTO pages (title, url, content, language) VALUES (?, ?, ?, ?)`,
			p.title, p.url, p.content, p.language)
		if err != nil {
			t.Fatalf("seed page %q: %v", p.title, err)
		}
	}

	tests := []struct {
		name       string
		q          string
		language   string
		wantTitles []string
	}{
		{name: "matches title", q: "Go language", language: "en", wantTitles: []string{"Go language"}},
		{name: "matches content", q: "compiled", language: "en", wantTitles: []string{"Go language"}},
		{name: "case insensitive", q: "COMPILED", language: "en", wantTitles: []string{"Go language"}},
		{name: "percent is literal", q: "50%", language: "en", wantTitles: []string{"Sales report"}},
		{name: "filters by language", q: "sprog", language: "da", wantTitles: []string{"Go sprog"}},
		{name: "other language not returned", q: "sprog", language: "en", wantTitles: []string{}},
		{name: "no match", q: "zzz", language: "en", wantTitles: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := searchPages(t.Context(), tt.q, tt.language)
			if err != nil {
				t.Fatalf("searchPages(%q, %q): %v", tt.q, tt.language, err)
			}
			if results == nil {
				t.Fatal("results is nil, want an empty slice so JSON encodes [] and not null")
			}
			gotTitles := []string{}
			for _, r := range results {
				gotTitles = append(gotTitles, r.Title)
			}
			if !slices.Equal(gotTitles, tt.wantTitles) {
				t.Errorf("titles = %q, want %q", gotTitles, tt.wantTitles)
			}
		})
	}
}

func TestSearchPagesTruncatesDescription(t *testing.T) {
	conn := newTestDB(t)

	_, err := conn.ExecContext(t.Context(),
		`INSERT INTO pages (title, url, content, language) VALUES (?, ?, ?, ?)`,
		"Long page", "https://example.com/long", strings.Repeat("a", descriptionMaxLen+100), "en")
	if err != nil {
		t.Fatalf("seed long page: %v", err)
	}

	results, err := searchPages(t.Context(), "aaa", "en")
	if err != nil {
		t.Fatalf("searchPages: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if got := len(results[0].Description); got != descriptionMaxLen {
		t.Errorf("description length = %d, want %d", got, descriptionMaxLen)
	}
}
