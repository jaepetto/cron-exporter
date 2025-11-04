package dashboard

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jaepetto/cron-exporter/pkg/config"
	"github.com/jaepetto/cron-exporter/pkg/model"
	"github.com/sirupsen/logrus"
)

// EventType represents the type of SSE event
type EventType string

const (
	EventJobStatusChange EventType = "job-status-change"
	EventJobCreated      EventType = "job-created"
	EventJobUpdated      EventType = "job-updated"
	EventJobDeleted      EventType = "job-deleted"
	EventHeartbeat       EventType = "heartbeat"
)

// SSEEvent represents a server-sent event
type SSEEvent struct {
	Type EventType   `json:"type"`
	Data interface{} `json:"data"`
}

// JobStatusUpdate represents a job status change event
type JobStatusUpdate struct {
	JobID          int       `json:"job_id"`
	Name           string    `json:"name"`
	Host           string    `json:"host"`
	Status         string    `json:"status"`
	LastReportedAt time.Time `json:"last_reported_at"`
	IsFailure      bool      `json:"is_failure"`
}

// SSEClient represents a connected SSE client
type SSEClient struct {
	id       string
	ctx      context.Context
	cancel   context.CancelFunc
	events   chan SSEEvent
	ginCtx   *gin.Context
	lastPing time.Time
}

// Broadcaster manages server-sent events for real-time updates
type Broadcaster struct {
	config    *config.DashboardConfig
	logger    *logrus.Logger
	jobStore  *model.JobStore
	clients   map[string]*SSEClient
	clientsMu sync.RWMutex
	events    chan SSEEvent
	quit      chan struct{}
}

// NewBroadcaster creates a new SSE broadcaster
func NewBroadcaster(config *config.DashboardConfig, jobStore *model.JobStore, logger *logrus.Logger) *Broadcaster {
	b := &Broadcaster{
		config:   config,
		logger:   logger,
		jobStore: jobStore,
		clients:  make(map[string]*SSEClient),
		events:   make(chan SSEEvent, 100),
		quit:     make(chan struct{}),
	}

	go b.run()
	return b
}

// run starts the broadcaster event loop
func (b *Broadcaster) run() {
	ticker := time.NewTicker(time.Duration(b.config.SSEHeartbeat) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event := <-b.events:
			b.broadcast(event)
		case <-ticker.C:
			b.sendHeartbeat()
			b.cleanupStaleClients()
		case <-b.quit:
			b.closeAllClients()
			return
		}
	}
}

// AddClient adds a new SSE client
func (b *Broadcaster) AddClient(ctx *gin.Context) *SSEClient {
	if !b.config.SSEEnabled {
		return nil
	}

	b.clientsMu.Lock()
	defer b.clientsMu.Unlock()

	// Check if we've reached the maximum number of clients
	if len(b.clients) >= b.config.SSEMaxClients {
		b.logger.Warn("Maximum SSE clients reached, rejecting new connection")
		return nil
	}

	clientID := fmt.Sprintf("client_%d_%d", time.Now().UnixNano(), len(b.clients))
	clientCtx, cancel := context.WithTimeout(context.Background(), time.Duration(b.config.SSETimeout)*time.Second)

	client := &SSEClient{
		id:       clientID,
		ctx:      clientCtx,
		cancel:   cancel,
		events:   make(chan SSEEvent, 10),
		ginCtx:   ctx,
		lastPing: time.Now(),
	}

	b.clients[clientID] = client
	b.logger.WithField("client_id", clientID).Info("New SSE client connected")

	return client
}

// RemoveClient removes an SSE client
func (b *Broadcaster) RemoveClient(clientID string) {
	b.clientsMu.Lock()
	defer b.clientsMu.Unlock()

	if client, exists := b.clients[clientID]; exists {
		client.cancel()
		close(client.events)
		delete(b.clients, clientID)
		b.logger.WithField("client_id", clientID).Info("SSE client disconnected")
	}
}

// BroadcastJobStatusChange broadcasts a job status change event
func (b *Broadcaster) BroadcastJobStatusChange(job *model.Job, isFailure bool) {
	if !b.config.SSEEnabled {
		return
	}

	event := SSEEvent{
		Type: EventJobStatusChange,
		Data: JobStatusUpdate{
			JobID:          job.ID,
			Name:           job.Name,
			Host:           job.Host,
			Status:         job.Status,
			LastReportedAt: job.LastReportedAt,
			IsFailure:      isFailure,
		},
	}

	select {
	case b.events <- event:
	default:
		b.logger.Warn("Event channel full, dropping job status change event")
	}
}

// BroadcastJobCreated broadcasts a job created event
func (b *Broadcaster) BroadcastJobCreated(job *model.Job) {
	if !b.config.SSEEnabled {
		return
	}

	event := SSEEvent{
		Type: EventJobCreated,
		Data: job,
	}

	select {
	case b.events <- event:
	default:
		b.logger.Warn("Event channel full, dropping job created event")
	}
}

// BroadcastJobUpdated broadcasts a job updated event
func (b *Broadcaster) BroadcastJobUpdated(job *model.Job) {
	if !b.config.SSEEnabled {
		return
	}

	event := SSEEvent{
		Type: EventJobUpdated,
		Data: job,
	}

	select {
	case b.events <- event:
	default:
		b.logger.Warn("Event channel full, dropping job updated event")
	}
}

// BroadcastJobDeleted broadcasts a job deleted event
func (b *Broadcaster) BroadcastJobDeleted(jobID int, name, host string) {
	if !b.config.SSEEnabled {
		return
	}

	event := SSEEvent{
		Type: EventJobDeleted,
		Data: map[string]interface{}{
			"job_id": jobID,
			"name":   name,
			"host":   host,
		},
	}

	select {
	case b.events <- event:
	default:
		b.logger.Warn("Event channel full, dropping job deleted event")
	}
}

// broadcast sends an event to all connected clients
func (b *Broadcaster) broadcast(event SSEEvent) {
	b.clientsMu.RLock()
	defer b.clientsMu.RUnlock()

	for clientID, client := range b.clients {
		select {
		case client.events <- event:
		default:
			b.logger.WithField("client_id", clientID).Warn("Client event channel full, dropping event")
		}
	}
}

// sendHeartbeat sends heartbeat events to all clients
func (b *Broadcaster) sendHeartbeat() {
	event := SSEEvent{
		Type: EventHeartbeat,
		Data: map[string]interface{}{
			"timestamp": time.Now(),
		},
	}

	b.broadcast(event)
}

// cleanupStaleClients removes clients that haven't been active
func (b *Broadcaster) cleanupStaleClients() {
	b.clientsMu.Lock()
	defer b.clientsMu.Unlock()

	staleTimeout := time.Duration(b.config.SSETimeout) * time.Second
	now := time.Now()

	for clientID, client := range b.clients {
		if now.Sub(client.lastPing) > staleTimeout {
			b.logger.WithField("client_id", clientID).Info("Removing stale SSE client")
			client.cancel()
			close(client.events)
			delete(b.clients, clientID)
		}
	}
}

// closeAllClients closes all connected clients
func (b *Broadcaster) closeAllClients() {
	b.clientsMu.Lock()
	defer b.clientsMu.Unlock()

	for clientID, client := range b.clients {
		b.logger.WithField("client_id", clientID).Info("Closing SSE client")
		client.cancel()
		close(client.events)
	}

	b.clients = make(map[string]*SSEClient)
}

// Stop stops the broadcaster
func (b *Broadcaster) Stop() {
	close(b.quit)
}

// GetStats returns broadcaster statistics
func (b *Broadcaster) GetStats() map[string]interface{} {
	b.clientsMu.RLock()
	defer b.clientsMu.RUnlock()

	return map[string]interface{}{
		"connected_clients": len(b.clients),
		"max_clients":       b.config.SSEMaxClients,
		"sse_enabled":       b.config.SSEEnabled,
	}
}

// ServeSSE handles the SSE connection for a client (simplified)
func (b *Broadcaster) ServeSSE(client *SSEClient) {
	// This method is now handled directly in the handler
	// Keep for compatibility but don't use
}
