package jobservice

import (
	"path/filepath"
	"testing"

	"github.com/jaepetto/cron-exporter/pkg/model"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) *Service {
	t.Helper()

	database, err := model.NewDatabase(filepath.Join(t.TempDir(), "jobservice-test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	return New(model.NewJobStore(database.GetDB()))
}

func TestCreateAppliesDefaults(t *testing.T) {
	service := newTestService(t)

	job, err := service.Create(CreateJobInput{Name: " backup ", Host: " db01 "})

	require.NoError(t, err)
	require.Equal(t, "backup", job.Name)
	require.Equal(t, "db01", job.Host)
	require.Equal(t, defaultFailureThreshold, job.AutomaticFailureThreshold)
	require.Equal(t, "active", job.Status)
	require.NotEmpty(t, job.ApiKey)
	require.NotNil(t, job.Labels)
	require.False(t, job.LastReportedAt.IsZero())
}

func TestCreateRejectsInvalidFields(t *testing.T) {
	service := newTestService(t)

	_, err := service.Create(CreateJobInput{
		AutomaticFailureThreshold: -1,
		Status:                    "unknown",
	})

	var serviceError *Error
	require.ErrorAs(t, err, &serviceError)
	require.Equal(t, ErrorValidation, serviceError.Kind)
	require.Equal(t, "is required", serviceError.Fields["job_name"])
	require.Equal(t, "is required", serviceError.Fields["host"])
	require.Contains(t, serviceError.Fields, "automatic_failure_threshold")
	require.Contains(t, serviceError.Fields, "status")
}

func TestUpdateAndToggleMaintenance(t *testing.T) {
	service := newTestService(t)
	job, err := service.Create(CreateJobInput{Name: "backup", Host: "db01"})
	require.NoError(t, err)

	name := "nightly-backup"
	threshold := 7200
	labels := map[string]string{"env": "prod"}
	updated, err := service.Update(job.ID, UpdateJobInput{
		Name:                      &name,
		AutomaticFailureThreshold: &threshold,
		Labels:                    &labels,
	})
	require.NoError(t, err)
	require.Equal(t, name, updated.Name)
	require.Equal(t, threshold, updated.AutomaticFailureThreshold)
	require.Equal(t, labels, updated.Labels)

	toggled, err := service.ToggleMaintenance(job.ID)
	require.NoError(t, err)
	require.Equal(t, "maintenance", toggled.Status)

	toggled, err = service.ToggleMaintenance(job.ID)
	require.NoError(t, err)
	require.Equal(t, "active", toggled.Status)
}

func TestDeleteReturnsNotFound(t *testing.T) {
	service := newTestService(t)

	_, err := service.Delete(999)

	require.True(t, IsKind(err, ErrorNotFound))
}

func TestListBoundsPageSize(t *testing.T) {
	service := newTestService(t)

	_, err := service.List(model.JobSearchCriteria{PageSize: maximumPageSize + 1})

	require.True(t, IsKind(err, ErrorValidation))
}

func TestListReturnsEmptyArray(t *testing.T) {
	service := newTestService(t)

	result, err := service.List(model.JobSearchCriteria{})

	require.NoError(t, err)
	require.NotNil(t, result.Jobs)
	require.Empty(t, result.Jobs)
}

func TestListPaginatesAfterLabelFiltering(t *testing.T) {
	service := newTestService(t)
	_, err := service.Create(CreateJobInput{
		Name:   "production-backup",
		Host:   "db01",
		Labels: map[string]string{"env": "prod"},
	})
	require.NoError(t, err)
	_, err = service.Create(CreateJobInput{
		Name:   "test-backup",
		Host:   "db02",
		Labels: map[string]string{"env": "test"},
	})
	require.NoError(t, err)

	result, err := service.List(model.JobSearchCriteria{
		Labels:   map[string]string{"env": "prod"},
		Page:     1,
		PageSize: 1,
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.TotalCount)
	require.Equal(t, 1, result.TotalPages)
	require.Len(t, result.Jobs, 1)
	require.Equal(t, "production-backup", result.Jobs[0].Name)
}
