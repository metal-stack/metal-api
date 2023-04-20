//go:build integration
// +build integration

package auditing_test

import (
	"context"
	"testing"

	"github.com/meilisearch/meilisearch-go"
	"github.com/metal-stack/metal-api/test"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"gotest.tools/assert"
)

func TestAuditing_Meilisearch(t *testing.T) {
	container, c, err := test.StartMeilisearch(t)
	require.NoError(t, err)
	defer func() {
		err := container.Terminate(context.Background())
		require.NoError(t, err)
	}()

	var (
		url = "http://" + c.IP + ":" + c.Port
	)

	auditing, err := New(Config{
		URL:              url,
		APIKey:           c.Password,
		IndexPrefix:      "test",
		RotationInterval: "",
		Log:              zaptest.NewLogger(t).Sugar(),
	})
	require.NoError(t, err)

	meiliAuditing := auditing.(*meiliAuditing)

	task, err := meiliAuditing.add(
		"rqid", rqID,
	)
	require.NoError(t, err)

	finalTask, _ := meiliAuditing.index.WaitForTask(task.TaskUID)
	require.Equal(t, meilisearch.TaskStatusSucceeded, finalTask.Status)

	result, err := meiliAuditing.search(rqID)
	require.NoError(t, err)

	require.Len(t, result, 1)
	assert.Equal(t, rqID, result[0].RequestID)
}
