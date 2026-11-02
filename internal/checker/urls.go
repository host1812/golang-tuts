package checker

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// LoadURLs reads one HTTP or HTTPS URL per line. Blank lines and lines that
// start with # are ignored. Invalid input fails at startup instead of producing
// confusing errors on every request.
func LoadURLs(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var targets []string
	scanner := bufio.NewScanner(file)
	for line := 1; scanner.Scan(); line++ {
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		u, err := url.Parse(text)
		if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
			return nil, fmt.Errorf("line %d: expected an HTTP(S) URL without credentials", line)
		}
		targets = append(targets, text)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read URL list: %w", err)
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("URL list is empty")
	}
	return targets, nil
}
