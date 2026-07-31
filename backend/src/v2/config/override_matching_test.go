package config

import "testing"

func TestPrefixMatchesOverride(t *testing.T) {
	tests := []struct {
		name        string
		prefix      string
		override    string
		shouldMatch bool
	}{
		{name: "exact match", prefix: "some/path/a/", override: "some/path/a", shouldMatch: true},
		{name: "nested path", prefix: "some/path/a/child/", override: "some/path/a", shouldMatch: true},
		{name: "trailing slash on override", prefix: "some/path/a/child/", override: "some/path/a/", shouldMatch: true},
		{name: "sibling path does not match", prefix: "some/path/allowed-other/", override: "some/path/allowed", shouldMatch: false},
		{name: "bucket-wide override", prefix: "any/path", override: "", shouldMatch: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if matched := prefixMatchesOverride(test.prefix, test.override); matched != test.shouldMatch {
				t.Fatalf("prefixMatchesOverride(%q, %q) = %v, want %v", test.prefix, test.override, matched, test.shouldMatch)
			}
		})
	}
}
