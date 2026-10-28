package main

import (
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
