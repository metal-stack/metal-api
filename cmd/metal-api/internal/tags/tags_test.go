package tags

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Example() {
	t := New(nil)
	t.Add("k=1")
	t.Add("k=2")
	fmt.Println(t.Unique())
	fmt.Println(t.Values("k="))
	// Output:
	// [k=1 k=2]
	// [1 2]
}

func TestHas(t *testing.T) {
	tests := []struct {
		name   string
		tags   []string
		tag    string
		wanted bool
	}{
		{
			name:   "empty",
			tags:   []string{},
			tag:    "",
			wanted: false,
		},
		{
			name:   "with tag",
			tags:   []string{"t"},
			tag:    "t",
			wanted: true,
		},
		{
			name:   "with other tags",
			tags:   []string{"a", "b", "c"},
			tag:    "t",
			wanted: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			tags := New(tt.tags)
			got := tags.Has(tt.tag)
			if !cmp.Equal(got, tt.wanted) {
				t.Errorf("Test failed: %v", cmp.Diff(got, tt.wanted))
			}
		})
	}
}

func TestHasPrefix(t *testing.T) {
	tests := []struct {
		name   string
		tags   []string
		prefix string
		wanted bool
	}{
		{
			name:   "empty tags",
			tags:   []string{},
			prefix: "",
			wanted: false,
		},
		{
			name:   "tag with empty string",
			tags:   []string{""},
			prefix: "",
			wanted: true,
		},
		{
			name:   "a tag with prefix",
			tags:   []string{"b", "c", "key=value"},
			prefix: "key",
			wanted: true,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			tags := New(tt.tags)
			got := tags.HasPrefix(tt.prefix)
			if !cmp.Equal(got, tt.wanted) {
				t.Errorf("Test failed: %v", cmp.Diff(got, tt.wanted))
			}
		})
	}
}

func TestRemove(t *testing.T) {
	tests := []struct {
		name         string
		tags         []string
		delete       string
		wantedTags   []string
		wantedReturn bool
	}{
		{
			name:         "tag not there",
			tags:         []string{""},
			delete:       "test",
			wantedTags:   []string{""},
			wantedReturn: false,
		},
		{
			name:         "remove a tag",
			tags:         []string{"2", "1", "2", "3"},
			delete:       "2",
			wantedTags:   []string{"1", "3"},
			wantedReturn: true,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			tags := New(tt.tags)
			gotReturn := tags.Remove(tt.delete)
			got := tags.Unique()
			if !cmp.Equal(got, tt.wantedTags) {
				t.Errorf("Test failed: %v", cmp.Diff(got, tt.wantedTags))
			}
			if gotReturn != tt.wantedReturn {
				t.Errorf("expected %v but got %v", tt.wantedReturn, gotReturn)
			}
		})
	}
}

func TestUnique(t *testing.T) {
	tests := []struct {
		name   string
		tags   []string
		wanted []string
	}{
		{
			name:   "empty",
			tags:   []string{},
			wanted: []string{},
		},
		{
			name:   "some tags",
			tags:   []string{"2", "1", "2"},
			wanted: []string{"1", "2"},
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			tags := New(tt.tags)
			got := tags.Unique()
			if !cmp.Equal(got, tt.wanted) {
				t.Errorf("Test failed: %v", cmp.Diff(got, tt.wanted))
			}
		})
	}
}
