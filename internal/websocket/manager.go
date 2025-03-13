package websocket

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	UploadProgress   NotificationType = "upload_progress"
	ProcessComplete  NotificationType = "process_complete"
	ProcessError     NotificationType = "process_error"
	UploadComplete   NotificationType = "upload_complete"
	ProcessingStatus NotificationType = "processing_status"
)

// Notification represents a WebSocket notification
type Notification struct {
	Type     NotificationType       `json:"type"`
	UserID   uint                   `json:"user_id"`
	MediaID  string                 `json:"media_id,omitempty"`
	Progress int                    `json:"progress,omitempty"`
	Message  string                 `json:"message,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
}

// Client represents a WebSocket client connection
type Client struct {
	UserID uint
	Conn   *websocket.Conn
}

// Manager handles WebSocket connections and notifications
type Manager struct {
	clients    map[uint][]*Client
	mu         sync.RWMutex
	register   chan *Client
	unregister chan *Client
}

var (
	instance *Manager
	once     sync.Once
)

// GetManager returns the singleton WebSocket manager instance
func GetManager() *Manager {
	once.Do(func() {
		instance = &Manager{
			clients:    make(map[uint][]*Client),
			register:   make(chan *Client),
			unregister: make(chan *Client),
		}
		go instance.run()
	})
	return instance
}

// run starts the WebSocket manager
func (m *Manager) run() {
	for {
		select {
		case client := <-m.register:
			m.mu.Lock()
			m.clients[client.UserID] = append(m.clients[client.UserID], client)
			m.mu.Unlock()
		case client := <-m.unregister:
			m.mu.Lock()
			if clients, ok := m.clients[client.UserID]; ok {
				for i, c := range clients {
					if c == client {
						m.clients[client.UserID] = append(clients[:i], clients[i+1:]...)
						break
					}
				}
				if len(m.clients[client.UserID]) == 0 {
					delete(m.clients, client.UserID)
				}
			}
			m.mu.Unlock()
		}
	}
}

// RegisterClient registers a new WebSocket client
func (m *Manager) RegisterClient(client *Client) {
	m.register <- client
}

// UnregisterClient unregisters a WebSocket client
func (m *Manager) UnregisterClient(client *Client) {
	m.unregister <- client
}

// SendNotification sends a notification to a specific user
func (m *Manager) SendNotification(userID uint, notification *Notification) error {
	m.mu.RLock()
	clients, ok := m.clients[userID]
	m.mu.RUnlock()

	if !ok {
		return nil // No clients connected for this user
	}

	data, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	for _, client := range clients {
		if err := client.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
			// Handle error but continue sending to other clients
			continue
		}
	}

	return nil
}

// SendUploadProgress sends an upload progress notification
func (m *Manager) SendUploadProgress(userID uint, mediaID string, progress int) {
	notification := &Notification{
		Type:     UploadProgress,
		UserID:   userID,
		MediaID:  mediaID,
		Progress: progress,
	}
	m.SendNotification(userID, notification)
}

// SendProcessingStatus sends a processing status notification
func (m *Manager) SendProcessingStatus(userID uint, mediaID string, status string) {
	notification := &Notification{
		Type:    ProcessingStatus,
		UserID:  userID,
		MediaID: mediaID,
		Message: status,
	}
	m.SendNotification(userID, notification)
}

// SendUploadComplete sends an upload complete notification
func (m *Manager) SendUploadComplete(userID uint, mediaID string, data map[string]interface{}) {
	notification := &Notification{
		Type:    UploadComplete,
		UserID:  userID,
		MediaID: mediaID,
		Data:    data,
	}
	m.SendNotification(userID, notification)
}

// SendProcessError sends a process error notification
func (m *Manager) SendProcessError(userID uint, mediaID string, errorMsg string) {
	notification := &Notification{
		Type:    ProcessError,
		UserID:  userID,
		MediaID: mediaID,
		Message: errorMsg,
	}
	m.SendNotification(userID, notification)
}
