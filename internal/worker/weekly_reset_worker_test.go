package worker_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/worker"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestWeeklyResetWorker_StartShutdown(t *testing.T) {
	ctx := context.Background()
	mockQuestSvc := mocks.NewMockQuestService(t)

	w := worker.NewWeeklyResetWorker(mockQuestSvc)
	require.NotNil(t, w)

	w.Start()

	w.Shutdown(ctx)
}
