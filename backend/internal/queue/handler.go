package queue

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)


var upgrader = websocket.Upgrader {
	ReadBufferSize: 1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {return true},
}

// QueueHandler handles HTTP requests for queue operations
type QueueHandler struct {
	service *QueueService
	connectionManager *ConnectionManager
}

// NewQueueHandler creates a new QueueHandler with the given service
func NewQueueHandler(service *QueueService, connectionManager *ConnectionManager) *QueueHandler {
	return &QueueHandler{
		service: service,
		connectionManager: connectionManager,
	}
}

func (h *QueueHandler) QueueWebSocketHandler(w http.ResponseWriter, r *http.Request) {
	fromIDStr := r.URL.Query().Get("from")
	toIDStr := r.URL.Query().Get("to")

	if fromIDStr == "" || toIDStr == "" {
		http.Error(w, "from and to terminal IDs are required", http.StatusBadRequest)
		return
	}

	fromID, err := uuid.Parse(fromIDStr)
	if err != nil {
		http.Error(w, "Invalid 'from' terminal ID format", http.StatusBadRequest)
		return
	}

	toID, err := uuid.Parse(toIDStr)
	if err != nil {
		http.Error(w, "Invalid 'to' terminal ID format", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	routeKey := fromID.String()+":"+toID.String()

	h.connectionManager.AddSubscriber(routeKey, conn)

	defer h.connectionManager.RemoveSubscribers(routeKey, conn)

	estimate, err := h.service.GetEstimate(r.Context(), fromID, toID)
	if err != nil {
		// If we can't get the initial estimate, still keep connection alive for updates
		// In production, you might want to send an error message or close the connection
	} else {
		// Send initial estimate
		initialMsg := map[string]interface{}{
			"type":           "estimate",
			"from_terminal":  fromID.String(),
			"to_terminal":    toID.String(),
			"estimate":       estimate,
			"timestamp":      time.Now().Unix(),
		}
		if err := conn.WriteJSON(initialMsg); err != nil {
			// If we can't write the initial message, connection is likely broken
			return
		}
	}

	// Keep connection alive to receive updates
	// In production, enhance this loop with ping/pong handling and proper timeout enforcement
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			// Client disconnected or error occurred
			break
		}
		// Optional: Handle pings or client messages here
		time.Sleep(1 * time.Second) // Prevent busy loop
	}
	
}

// StoreReportHandler handles POST /queue/report
func (h *QueueHandler) StoreReportHandler(w http.ResponseWriter, r *http.Request) {
	var report QueueReport
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Set the reported timestamp to now if not provided (should come from client)
	if report.ReportedAt.IsZero() {
		report.ReportedAt = time.Now()
	}

	id, err := h.service.StoreReport(r.Context(), report)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidReport):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrSameTerminals):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrTerminalNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, "Internal server error: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": id,
	})
}

// GetEstimateHandler handles GET /queue/estimate?from=&to=
func (h *QueueHandler) GetEstimateHandler(w http.ResponseWriter, r *http.Request) {
	// Extract IDs from query parameters
	fromIDStr := r.URL.Query().Get("from")
	toIDStr := r.URL.Query().Get("to")

	if fromIDStr == "" || toIDStr == "" {
		http.Error(w, "from and to terminal IDs are required", http.StatusBadRequest)
		return
	}

	fromID, err := uuid.Parse(fromIDStr)
	if err != nil {
		http.Error(w, "Invalid 'from' terminal ID format", http.StatusBadRequest)
		return
	}

	toID, err := uuid.Parse(toIDStr)
	if err != nil {
		http.Error(w, "Invalid 'to' terminal ID format", http.StatusBadRequest)
		return
	}

	estimate, err := h.service.GetEstimate(r.Context(), fromID, toID)
	if err != nil {
		switch {
		case errors.Is(err, ErrNoRecentData):
			// Return 200 with explicit no-estimate when successful but no data
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"from_terminal_id": fromID,
				"to_terminal_id":   toID,
				"message":          "no recent wait-time data available",
			})
			return
		default:
			http.Error(w, "Internal server error: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(estimate)
}