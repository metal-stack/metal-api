package tags

import (
	"sort"
	"strings"
)

// Tags holds tags.
type Tags struct {
	tags []string
}

// New creates a new Tag instance.
func New(tags []string) *Tags {
	return &Tags{
		tags: tags,
	}
}

// Has checks whether the given tag is contained in the tags.
func (t *Tags) Has(tag string) bool {
	for _, t := range t.tags {
		if t == tag {
			return true
		}
	}
	return false
}

// HasPrefix checks whether the given prefix is contained in the tags.
func (t *Tags) HasPrefix(prefix string) bool {
	for _, t := range t.tags {
		if strings.HasPrefix(t, prefix) {
			return true
		}
	}
	return false
}

// Add adds a tag
func (t *Tags) Add(tag string) {
	t.tags = append(t.tags, tag)
}

// Remove removes a tag
func (t *Tags) Remove(tag string) bool {
	var tags []string
	removed := false
	for _, t := range t.tags {
		if t == tag {
			removed = true
			continue
		}
		tags = append(tags, t)
	}
	if removed {
		t.tags = tags
	}
	return removed
}

// Values collects all the values that are contained with the given prefix.
func (t *Tags) Values(prefix string) []string {
	var values []string
	for _, t := range t.tags {
		if strings.HasPrefix(t, prefix) {
			values = append(values, strings.TrimPrefix(t, prefix))
		}
	}
	return values
}

// Unique returns the distinct tag values as sorted slice.
func (t *Tags) Unique() []string {
	tagSet := make(map[string]bool)
	for _, t := range t.tags {
		tagSet[t] = true
	}
	var uniqueTags []string
	for k := range tagSet {
		uniqueTags = append(uniqueTags, k)
	}
	sort.Strings(uniqueTags)
	return uniqueTags
}
