package tags

import (
	"sort"
	"strings"
)

type Tags struct {
	tags []string
}

func New(tags []string) *Tags {
	return &Tags{
		tags: tags,
	}
}

func (t *Tags) Has(tag string) bool {
	for _, t := range t.tags {
		if t == tag {
			return true
		}
	}
	return false
}

func (t *Tags) HasPrefix(prefix string) bool {
	for _, t := range t.tags {
		if strings.HasPrefix(t, prefix) {
			return true
		}
	}
	return false
}

func (t *Tags) Add(tag string) {
	t.tags = append(t.tags, tag)
}

func (t *Tags) Remove(tag string) bool {
	tags := []string{}
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

func (t *Tags) Values(prefix string) []string {
	values := []string{}
	for _, t := range t.tags {
		if strings.HasPrefix(t, prefix) {
			values = append(values, strings.TrimPrefix(t, prefix))
		}
	}
	return values
}

func (t *Tags) ClearValue(tag, seperator string) {
	r := []string{}
	for _, t := range t.tags {
		if t == tag {
			s := strings.Split(t, seperator)
			r = append(r, s[0])
			continue
		}
		r = append(r, t)
	}
	t.tags = r
}

func (t *Tags) Unique() []string {
	tagSet := make(map[string]bool)
	for _, t := range t.tags {
		tagSet[t] = true
	}
	uniqueTags := []string{}
	for k := range tagSet {
		uniqueTags = append(uniqueTags, k)
	}
	sort.Strings(uniqueTags)
	return uniqueTags
}
