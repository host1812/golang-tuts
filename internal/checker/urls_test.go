package checker

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadURLs(t *testing.T) {
	tests := []struct {
		name, input string
		want        []string
		wantErr     bool
	}{
		{"valid", "# comment\n https://example.com \n\nhttp://example.org/path\n", []string{"https://example.com", "http://example.org/path"}, false},
		{"invalid scheme", "ftp://example.com\n", nil, true},
		{"credentials", "https://user:pass@example.com\n", nil, true},
		{"empty", "# comment\n", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "list")
			if err := os.WriteFile(path, []byte(tt.input), 0600); err != nil {
				t.Fatal(err)
			}
			got, err := LoadURLs(path)
			if (err != nil) != tt.wantErr || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, error %v", got, err)
			}
		})
	}
	if _, err := LoadURLs(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("expected missing file error")
	}
}
