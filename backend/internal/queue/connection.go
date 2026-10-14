package queue

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type ConnectionManager struct {
	subscribers map[string][]*websocket.Conn
	mutex       sync.RWMutex
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		subscribers: make(map[string][]*websocket.Conn),
	}
}

func (cm *ConnectionManager) AddSubscriber(routeKey string, conn *websocket.Conn) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.subscribers[routeKey] = append(cm.subscribers[routeKey], conn)
}

func (cm *ConnectionManager) RemoveSubscribers(routeKey string, conn *websocket.Conn) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	if conns, exists := cm.subscribers[routeKey]; exists {
		for i, c := range conns {
			if c == conn {
				cm.subscribers[routeKey] = append(conns[:i], conns[i+1:]...)
				if len(cm.subscribers[routeKey]) == 0 {
					delete(cm.subscribers, routeKey)
				}
				return
			}
		}
	}
}

func (cm *ConnectionManager) BroadcastToRoute(routeKey string, message interface{}) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if conns, exists := cm.subscribers[routeKey]; exists {
		for _, conn := range conns {
			err := conn.WriteJSON(message)
			if err != nil {
				cm.RemoveSubscribers(routeKey, conn)
			}
		}
	}
}

func (cm *ConnectionManager) GetSubscriberCount(routeKey string) int {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if conns, exists := cm.subscribers[routeKey]; exists {
		return len(conns)
	}
	return 0
}

// SimpleQueueNotifier is a basic implementation that forwards notifications to the connection manager
type SimpleQueueNotifier struct {
	ConnMgr *ConnectionManager
}

// NotifyQueueUpdate forwards the queue update notification to the connection manager
func (n *SimpleQueueNotifier) NotifyQueueUpdate(fromID, toID uuid.UUID) {
	routeKey := fromID.String() + ":" + toID.String()

	// Create a lightweight notification message
	updateMsg := map[string]interface{}{
		"type":           "queue_update",
		"from_terminal":  fromID.String(),
		"to_terminal":    toID.String(),
		"timestamp":      time.Now().Unix(),
	}

	// Broadcast to all subscribers of this route
	n.ConnMgr.BroadcastToRoute(routeKey, updateMsg)
}