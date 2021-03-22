package service

import (
	"github.com/stretchr/testify/require"
	"sort"
	"testing"
)

func TestInsertRevisions(t *testing.T) {
	// given
	paths := []string{
		"bucket/v/b/vb1",
		"bucket/v/b/vb2",
		"bucket/v/b/vb3",
		"bucket/v/c/vc1",
		"bucket/v/c/vc2",
		"bucket/x/y/xy1",
		"bucket/x/y/xy2",
	}
	revisions := make(map[string]map[string][]string)

	// when
	for _, path := range paths {
		insertRevisions(path, revisions, "v", "b")
	}

	// then
	require.Equal(t, 1, len(revisions))
	boardRevisions, ok := revisions["v"]
	require.True(t, ok)
	require.Equal(t, 1, len(boardRevisions))
	rr, ok := boardRevisions["b"]
	require.True(t, ok)
	sort.Strings(rr)
	require.Equal(t, []string{"vb1", "vb2", "vb3"}, rr)

	// given
	revisions = make(map[string]map[string][]string)

	// when
	for _, path := range paths {
		insertRevisions(path, revisions, "", "b")
	}

	// then
	require.Equal(t, 1, len(revisions))
	boardRevisions, ok = revisions["v"]
	require.True(t, ok)
	require.Equal(t, 1, len(boardRevisions))
	rr, ok = boardRevisions["b"]
	require.True(t, ok)
	sort.Strings(rr)
	require.Equal(t, []string{"vb1", "vb2", "vb3"}, rr)

	// when
	for _, path := range paths {
		insertRevisions(path, revisions, "v", "")
	}

	// then
	require.Equal(t, 1, len(revisions))
	boardRevisions, ok = revisions["v"]
	require.True(t, ok)
	require.Equal(t, 2, len(boardRevisions))
	rr, ok = boardRevisions["b"]
	require.True(t, ok)
	sort.Strings(rr)
	require.Equal(t, []string{"vb1", "vb2", "vb3"}, rr)
	rr, ok = boardRevisions["c"]
	require.True(t, ok)
	sort.Strings(rr)
	require.Equal(t, []string{"vc1", "vc2"}, rr)

	// when
	for _, path := range paths {
		insertRevisions(path, revisions, "", "")
	}

	// then
	require.Equal(t, 2, len(revisions))
	boardRevisions, ok = revisions["v"]
	require.True(t, ok)
	require.Equal(t, 2, len(boardRevisions))
	rr, ok = boardRevisions["b"]
	require.True(t, ok)
	sort.Strings(rr)
	require.Equal(t, []string{"vb1", "vb2", "vb3"}, rr)
	rr, ok = boardRevisions["c"]
	require.True(t, ok)
	sort.Strings(rr)
	require.Equal(t, []string{"vc1", "vc2"}, rr)

	boardRevisions, ok = revisions["x"]
	require.True(t, ok)
	require.Equal(t, 1, len(boardRevisions))
	rr, ok = boardRevisions["y"]
	require.True(t, ok)
	sort.Strings(rr)
	require.Equal(t, []string{"xy1", "xy2"}, rr)
}
