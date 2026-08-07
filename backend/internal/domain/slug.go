package domain

import "github.com/gosimple/slug"

// Slugify derives a URL-safe slug from a Company name — transliterating
// accents (e.g. "Logística" -> "logistica") rather than dropping them. The
// slug is never typed by the consultant — this is the only place it comes
// from.
func Slugify(name string) string {
	s := slug.Make(name)
	if s == "" {
		s = "company"
	}
	return s
}
