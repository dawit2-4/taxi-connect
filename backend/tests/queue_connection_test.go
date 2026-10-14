package tests

import (
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"taxi-connect/backend/internal/queue"
)

func TestConnectionManager_AddSubscriber(t *testing.T) {
	cm := queue.NewConnectionManager()
	conn := &websocket.Conn{}

	routeKey := "test-route"
	cm.AddSubscriber(routeKey, conn)

	count := cm.GetSubscriberCount(routeKey)
	if count != 1 {
		t.Errorf("Expected 1 subscriber, got %d", count)
	}
}

func TestConnectionManager_RemoveSubscribers(t *testing.T) {
	cm := queue.NewConnectionManager()
	conn := &websocket.Conn{}

	routeKey := "test-route"
	cm.AddSubscriber(routeKey, conn)
	cm.RemoveSubscribers(routeKey, conn)

	count := cm.GetSubscriberCount(routeKey)
	if count != 0 {
		t.Errorf("Expected 0 subscribers, got %d", count)
	}
}

func TestConnectionManager_BroadcastToRoute(t *testing.T) {
	cm := queue.NewConnectionManager()

	// Test broadcasting to a route with no subscribers (should not panic)
	message := map[string]interface{}{
		"type": "test",
	}
	cm.BroadcastToRoute("nonexistent-route", message)

	// Test that we can add and remove subscribers without panicking
	conn := &websocket.Conn{}
	cm.AddSubscriber("test-route", conn)
	if cm.GetSubscriberCount("test-route") != 1 {
		t.Errorf("Expected 1 subscriber after adding, got %d", cm.GetSubscriberCount("test-route"))
	}

	cm.RemoveSubscribers("test-route", conn)
	if cm.GetSubscriberCount("test-route") != 0 {
		t.Errorf("Expected 0 subscribers after removing, got %d", cm.GetSubscriberCount("test-route"))
	}

	// Test broadcasting again to a route with no subscribers (should not panic)
	cm.BroadcastToRoute("test-route", message)
}

func TestSimpleQueueNotifier_NotifyQueueUpdate(t *testing.T) {
	cm := queue.NewConnectionManager()
	notifier := &queue.SimpleQueueNotifier{ConnMgr: cm}

	fromID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	toID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	// This test mainly ensures the function doesn't panic
	// In a real test, we would mock the WebSocket connections to verify WriteJSON was called
	notifier.NotifyQueueUpdate(fromID, toID)

	// Verify that a subscriber entry was created for the route
	routeKey := fromID.String() + ":" + toID.String()
	count := cm.GetSubscriberCount(routeKey)
	if count != 0 {
		t.Errorf("Expected 0 subscribers (no actual connections added), got %d", count)
	}
}