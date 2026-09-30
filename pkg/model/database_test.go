package model

import (
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestDatabaseWaitsForConcurrentWriters(t *testing.T) {
	originalOutput := logrus.StandardLogger().Out
	logrus.SetOutput(io.Discard)
	t.Cleanup(func() { logrus.SetOutput(originalOutput) })

	database, err := NewDatabase(filepath.Join(t.TempDir(), "concurrent-writes.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	database.GetDB().SetMaxOpenConns(50)
	store := NewJobStore(database.GetDB())

	start := make(chan struct{})
	errors := make(chan error, 50)
	var writers sync.WaitGroup
	for index := 0; index < 50; index++ {
		writers.Add(1)
		go func(jobIndex int) {
			defer writers.Done()
			<-start
			errors <- store.CreateJob(&Job{
				Name:                      fmt.Sprintf("concurrent-%d", jobIndex),
				Host:                      "test-host",
				ApiKey:                    fmt.Sprintf("cm_concurrent_%d", jobIndex),
				AutomaticFailureThreshold: 3600,
				Labels:                    map[string]string{},
				Status:                    "active",
				LastReportedAt:            time.Now().UTC(),
			})
		}(index)
	}
	close(start)
	writers.Wait()
	close(errors)

	for err := range errors {
		require.NoError(t, err)
	}
}
