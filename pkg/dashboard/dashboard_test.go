package dashboard

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jaepetto/cron-exporter/pkg/config"
	"github.com/jaepetto/cron-exporter/pkg/model"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

const testAdminAPIKey = "test-admin-key"

func newTestDashboard(t *testing.T) (*Dashboard, *model.JobStore) {
	t.Helper()

	database, err := model.NewDatabase(filepath.Join(t.TempDir(), "dashboard-test.db"))
	require.NoError(t, err)
	jobStore := model.NewJobStore(database.GetDB())
	cfg := &config.DashboardConfig{
		Enabled:         true,
		Path:            "/dashboard",
		Title:           "Test Cron Monitor",
		PageSize:        25,
		AuthRequired:    true,
		SSEEnabled:      true,
		SSETimeout:      30,
		SSEHeartbeat:    30,
		SSEMaxClients:   2,
		PollingFallback: true,
		PollingInterval: 5,
	}

	dashboard := New(cfg, jobStore, []string{testAdminAPIKey}, logrus.New())
	t.Cleanup(func() {
		dashboard.handler.broadcaster.Stop()
		require.NoError(t, database.Close())
	})

	return dashboard, jobStore
}

func performDashboardRequest(t *testing.T, dashboard *Dashboard, method, target string, authenticated bool) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, target, nil)
	if authenticated {
		request.SetBasicAuth("admin", testAdminAPIKey)
	}

	response := httptest.NewRecorder()
	dashboard.Router().ServeHTTP(response, request)
	return response
}

func performDashboardJSONRequest(t *testing.T, dashboard *Dashboard, method, target string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()

	var encodedBody bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&encodedBody).Encode(body))
	}
	request := httptest.NewRequest(method, target, &encodedBody)
	request.Header.Set("Content-Type", "application/json")
	request.SetBasicAuth("admin", testAdminAPIKey)

	response := httptest.NewRecorder()
	dashboard.Router().ServeHTTP(response, request)
	return response
}

func TestDashboardRequiresBasicAuthentication(t *testing.T) {
	dashboard, _ := newTestDashboard(t)

	unauthenticated := performDashboardRequest(t, dashboard, http.MethodGet, "/jobs", false)
	require.Equal(t, http.StatusUnauthorized, unauthenticated.Code)
	require.Equal(t, `Basic realm="Dashboard"`, unauthenticated.Header().Get("WWW-Authenticate"))

	authenticated := performDashboardRequest(t, dashboard, http.MethodGet, "/jobs", true)
	require.Equal(t, http.StatusOK, authenticated.Code)
}

func TestDashboardAssetsDoNotRequireAuthentication(t *testing.T) {
	dashboard, _ := newTestDashboard(t)
	assetMatch := regexp.MustCompile(`\./assets/([^"']+\.js)`).FindSubmatch(dashboard.handler.portal.indexHTML)
	require.Len(t, assetMatch, 2)

	response := performDashboardRequest(t, dashboard, http.MethodGet, "/assets/"+string(assetMatch[1]), false)

	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Header().Get("Content-Type"), "javascript")
	require.Equal(t, "public, max-age=31536000, immutable", response.Header().Get("Cache-Control"))
}

func TestDashboardShellInjectsRuntimeConfigAndSupportsDeepLinks(t *testing.T) {
	dashboard, _ := newTestDashboard(t)

	response := performDashboardRequest(t, dashboard, http.MethodGet, "/jobs/42/edit", true)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "no-cache, no-store, must-revalidate", response.Header().Get("Cache-Control"))
	require.Contains(t, response.Body.String(), `content="/dashboard"`)
	require.Contains(t, response.Body.String(), "Test Cron Monitor")
	require.NotContains(t, response.Body.String(), "__CRONMETRICS_")
}

func TestJobStatusAPI(t *testing.T) {
	dashboard, jobStore := newTestDashboard(t)
	job := &model.Job{
		Name:                      "nightly-backup",
		Host:                      "db01",
		ApiKey:                    "cm_test_job_key",
		AutomaticFailureThreshold: 3600,
		Labels:                    map[string]string{"env": "test"},
		Status:                    "active",
		LastReportedAt:            time.Now().UTC(),
	}
	require.NoError(t, jobStore.CreateJob(job))

	response := performDashboardRequest(t, dashboard, http.MethodGet, "/api/jobs/"+strconv.Itoa(job.ID)+"/status", true)
	require.Equal(t, http.StatusOK, response.Code)

	var status JobStatusUpdate
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &status))
	require.Equal(t, job.ID, status.JobID)
	require.Equal(t, job.Name, status.Name)
	require.Equal(t, job.Host, status.Host)
	require.Equal(t, job.Status, status.Status)
	require.False(t, status.IsFailure)
}

func TestDashboardJobAPIWorkflow(t *testing.T) {
	dashboard, _ := newTestDashboard(t)

	createResponse := performDashboardJSONRequest(t, dashboard, http.MethodPost, "/api/jobs", map[string]interface{}{
		"job_name":                    "nightly-backup",
		"host":                        "db01",
		"automatic_failure_threshold": 1800,
		"labels":                      map[string]string{"env": "test"},
	})
	require.Equal(t, http.StatusCreated, createResponse.Code, createResponse.Body.String())
	var created model.Job
	require.NoError(t, json.Unmarshal(createResponse.Body.Bytes(), &created))
	require.Positive(t, created.ID)
	require.Equal(t, "active", created.Status)

	listResponse := performDashboardJSONRequest(t, dashboard, http.MethodGet, "/api/jobs?q=nightly&page=1&page_size=10", nil)
	require.Equal(t, http.StatusOK, listResponse.Code, listResponse.Body.String())
	var result model.JobSearchResult
	require.NoError(t, json.Unmarshal(listResponse.Body.Bytes(), &result))
	require.Equal(t, 1, result.TotalCount)
	require.Len(t, result.Jobs, 1)

	detailPath := "/api/jobs/" + strconv.Itoa(created.ID)
	detailResponse := performDashboardJSONRequest(t, dashboard, http.MethodGet, detailPath, nil)
	require.Equal(t, http.StatusOK, detailResponse.Code, detailResponse.Body.String())

	updateResponse := performDashboardJSONRequest(t, dashboard, http.MethodPut, detailPath, map[string]interface{}{
		"status": "paused",
	})
	require.Equal(t, http.StatusOK, updateResponse.Code, updateResponse.Body.String())
	var updated model.Job
	require.NoError(t, json.Unmarshal(updateResponse.Body.Bytes(), &updated))
	require.Equal(t, "paused", updated.Status)

	toggleResponse := performDashboardJSONRequest(t, dashboard, http.MethodPost, detailPath+"/toggle", nil)
	require.Equal(t, http.StatusOK, toggleResponse.Code, toggleResponse.Body.String())
	require.NoError(t, json.Unmarshal(toggleResponse.Body.Bytes(), &updated))
	require.Equal(t, "maintenance", updated.Status)

	deleteResponse := performDashboardJSONRequest(t, dashboard, http.MethodDelete, detailPath, nil)
	require.Equal(t, http.StatusNoContent, deleteResponse.Code, deleteResponse.Body.String())

	missingResponse := performDashboardJSONRequest(t, dashboard, http.MethodGet, detailPath, nil)
	require.Equal(t, http.StatusNotFound, missingResponse.Code, missingResponse.Body.String())
}

func TestDashboardJobAPIValidation(t *testing.T) {
	dashboard, _ := newTestDashboard(t)

	response := performDashboardJSONRequest(t, dashboard, http.MethodPost, "/api/jobs", map[string]interface{}{
		"job_name":                    "invalid-job",
		"host":                        "db01",
		"automatic_failure_threshold": -1,
		"status":                      "unknown",
	})

	require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
	var apiError APIError
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &apiError))
	require.Equal(t, "validation", apiError.Code)
	require.Contains(t, apiError.Fields, "automatic_failure_threshold")
	require.Contains(t, apiError.Fields, "status")
}

func TestWriteSSEMessageFlushesResponse(t *testing.T) {
	dashboard, _ := newTestDashboard(t)
	response := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(response)

	written := dashboard.handler.writeSSEMessage(ginContext, "job-updated", map[string]int{"job_id": 42})

	require.True(t, written)
	require.True(t, response.Flushed)
	require.Contains(t, response.Body.String(), "event: job-updated")
	require.Contains(t, response.Body.String(), `data: {"job_id":42}`)
}

func TestBroadcasterEnforcesClientLimit(t *testing.T) {
	dashboard, _ := newTestDashboard(t)

	first := addTestSSEClient(t, dashboard)
	second := addTestSSEClient(t, dashboard)
	third := addTestSSEClient(t, dashboard)

	require.NotNil(t, first)
	require.NotNil(t, second)
	require.Nil(t, third)
	require.Equal(t, 2, dashboard.handler.broadcaster.GetStats()["connected_clients"])
}

func TestBroadcasterDisconnectsSlowClient(t *testing.T) {
	dashboard, _ := newTestDashboard(t)
	client := addTestSSEClient(t, dashboard)
	require.NotNil(t, client)

	for index := 0; index < cap(client.events); index++ {
		client.events <- SSEEvent{Type: EventJobUpdated}
	}
	dashboard.handler.broadcaster.broadcast(SSEEvent{Type: EventJobUpdated})

	require.Equal(t, 0, dashboard.handler.broadcaster.GetStats()["connected_clients"])
}

func addTestSSEClient(t *testing.T, dashboard *Dashboard) *SSEClient {
	t.Helper()
	ginContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/events", nil)
	return dashboard.handler.broadcaster.AddClient(ginContext)
}
