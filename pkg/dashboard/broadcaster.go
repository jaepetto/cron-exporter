package dashboard

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
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
	EventConnected       EventType = "connected"
	EventReset           EventType = "reset"
)

// SSEEvent represents a cache invalidation sent to dashboard clients.
type SSEEvent struct {
	ID        string    `json:"id"`
	Type      EventType `json:"type"`
	JobID     *int      `json:"job_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
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
	id     string
	ctx    context.Context
	cancel context.CancelFunc
	events chan SSEEvent
	ginCtx *gin.Context
}

// Broadcaster manages server-sent events for real-time updates
type Broadcaster struct {
	config    *config.DashboardConfig
	logger    *logrus.Logger
	clients   map[string]*SSEClient
	clientsMu sync.RWMutex
	events    chan SSEEvent
	quit      chan struct{}
	done      chan struct{}
	stopOnce  sync.Once
	sequence  atomic.Uint64
}

// NewBroadcaster creates a new SSE broadcaster
func NewBroadcaster(config *config.DashboardConfig, logger *logrus.Logger) *Broadcaster {
	b := &Broadcaster{
		config:  config,
		logger:  logger,
		clients: make(map[string]*SSEClient),
		events:  make(chan SSEEvent, 100),
		quit:    make(chan struct{}),
		done:    make(chan struct{}),
	}

	go b.run()
	return b
}

// run starts the broadcaster event loop
func (b *Broadcaster) run() {
	defer close(b.done)
	ticker := time.NewTicker(time.Duration(b.config.SSEHeartbeat) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event := <-b.events:
			b.broadcast(event)
		case <-ticker.C:
			b.sendHeartbeat()
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
	clientCtx, cancel := context.WithTimeout(ctx.Request.Context(), time.Duration(b.config.SSETimeout)*time.Second)

	client := &SSEClient{
		id:     clientID,
		ctx:    clientCtx,
		cancel: cancel,
		events: make(chan SSEEvent, 10),
		ginCtx: ctx,
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
func (b *Broadcaster) BroadcastJobStatusChange(job *model.Job, _ bool) {
	if !b.config.SSEEnabled {
		return
	}

	b.enqueue(b.newEvent(EventJobStatusChange, &job.ID))
}

// BroadcastJobCreated broadcasts a job created event
func (b *Broadcaster) BroadcastJobCreated(job *model.Job) {
	if !b.config.SSEEnabled {
		return
	}

	b.enqueue(b.newEvent(EventJobCreated, &job.ID))
}

// BroadcastJobUpdated broadcasts a job updated event
func (b *Broadcaster) BroadcastJobUpdated(job *model.Job) {
	if !b.config.SSEEnabled {
		return
	}

	b.enqueue(b.newEvent(EventJobUpdated, &job.ID))
}

// BroadcastJobDeleted broadcasts a job deleted event
func (b *Broadcaster) BroadcastJobDeleted(jobID int, _, _ string) {
	if !b.config.SSEEnabled {
		return
	}

	b.enqueue(b.newEvent(EventJobDeleted, &jobID))
}

// broadcast sends an event to all connected clients
func (b *Broadcaster) broadcast(event SSEEvent) {
	b.clientsMu.RLock()
	var slowClientIDs []string
	for clientID, client := range b.clients {
		select {
		case client.events <- event:
		default:
			slowClientIDs = append(slowClientIDs, clientID)
		}
	}
	b.clientsMu.RUnlock()

	for _, clientID := range slowClientIDs {
		b.logger.WithField("client_id", clientID).Warn("Disconnecting slow SSE client for full revalidation")
		b.RemoveClient(clientID)
	}
}

// sendHeartbeat sends heartbeat events to all clients
func (b *Broadcaster) sendHeartbeat() {
	b.broadcast(b.newEvent(EventHeartbeat, nil))
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
	b.stopOnce.Do(func() { close(b.quit) })
	<-b.done
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

func (b *Broadcaster) newEvent(eventType EventType, jobID *int) SSEEvent {
	sequence := b.sequence.Add(1)
	now := time.Now().UTC()
	return SSEEvent{
		ID:        fmt.Sprintf("%d-%d", now.UnixMilli(), sequence),
		Type:      eventType,
		JobID:     jobID,
		Timestamp: now,
	}
}

func (b *Broadcaster) enqueue(event SSEEvent) {
	select {
	case b.events <- event:
	default:
		b.logger.Warn("SSE event queue full; disconnecting clients for full revalidation")
		b.closeAllClients()
	}
}
