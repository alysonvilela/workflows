package domain

import (
	"regexp"
	"strings"
)

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify derives a URL-safe slug from a Company name: lowercased, runs of
// non-alphanumeric characters collapsed to one hyphen, edges trimmed. The
// slug is never typed by the consultant — this is the only place it comes
// from.
func Slugify(name string) string {
	slug := nonSlugChars.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "company"
	}
	return slug
}
