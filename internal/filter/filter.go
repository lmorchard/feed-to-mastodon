// Package filter decides whether feed entries may be posted based on their
// categories.
package filter

import "strings"

// Filter applies include and exclude category rules to feed entries.
//
// Matching is case-insensitive and exact: a configured category must equal a
// whole entry category, not a substring of it.
type Filter struct {
	include []string
	exclude []string
}

// New creates a Filter from include and exclude category lists. Inputs are
// normalized: surrounding whitespace is trimmed, empty values are dropped, and
// values are deduplicated case-insensitively while preserving first-seen order.
func New(include, exclude []string) *Filter {
	return &Filter{
		include: normalize(include),
		exclude: normalize(exclude),
	}
}

// Enabled reports whether any category rule is configured. A disabled filter
// allows every entry.
func (f *Filter) Enabled() bool {
	return len(f.include) > 0 || len(f.exclude) > 0
}

// Allowed reports whether an entry with the given categories may be posted.
//
// Exclusions always win. When no include categories are configured, an entry
// is postable unless it matches an exclusion. When include categories are
// configured, an entry must match at least one of them.
func (f *Filter) Allowed(categories []string) bool {
	if !f.Enabled() {
		return true
	}

	for _, category := range categories {
		if matchesAny(category, f.exclude) {
			return false
		}
	}

	if len(f.include) == 0 {
		return true
	}

	for _, category := range categories {
		if matchesAny(category, f.include) {
			return true
		}
	}

	return false
}

// matchesAny reports whether category equals (case-insensitively) any value in
// categories.
func matchesAny(category string, categories []string) bool {
	category = strings.TrimSpace(category)
	if category == "" {
		return false
	}

	for _, candidate := range categories {
		if strings.EqualFold(category, candidate) {
			return true
		}
	}

	return false
}

// normalize trims values, drops empty strings, and removes case-insensitive
// duplicates while preserving first-seen order and casing.
func normalize(values []string) []string {
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}
		normalized = append(normalized, value)
	}

	return normalized
}
