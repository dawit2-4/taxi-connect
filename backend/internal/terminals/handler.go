package terminals

import (
	// "context"
	"encoding/json"
	"net/http"
	"strconv"
	"errors"

	"github.com/google/uuid"
)

// TerminalHandler handles HTTP requests for terminals
type TerminalHandler struct {
	service *TerminalService
}

// NewTerminalHandler creates a new TerminalHandler with the given service
func NewTerminalHandler(service *TerminalService) *TerminalHandler {
	return &TerminalHandler{service: service}
}

// CreateTerminalHandler handles POST /terminals
func (h *TerminalHandler) CreateTerminalHandler(w http.ResponseWriter, r *http.Request) {
	var terminal Terminal
	if err := json.NewDecoder(r.Body).Decode(&terminal); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Validate required fields
	if terminal.Name == "" {
		http.Error(w, "Terminal name is required", http.StatusBadRequest)
		return
	}

	id, err := h.service.CreateTerminal(r.Context(), terminal)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmptyName):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrNameTooLong):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrDuplicateTerminalName):
			http.Error(w, err.Error(), http.StatusConflict)
		case errors.Is(err, ErrInvalidLocation):
			http.Error(w, err.Error(), http.StatusBadRequest)
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

// GetTerminalHandler handles GET /terminals/{id}
func (h *TerminalHandler) GetTerminalHandler(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	pathParams := r.URL.Path[len("/terminals/"):]
	if pathParams == "" {
		http.Error(w, "Terminal ID is required", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(pathParams)
	if err != nil {
		http.Error(w, "Invalid terminal ID format", http.StatusBadRequest)
		return
	}

	terminal, err := h.service.GetTerminal(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, ErrTerminalNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, "Internal server error: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(terminal)
}

// GetTerminalByNameHandler handles GET /terminals/name/{name}
func (h *TerminalHandler) GetTerminalByNameHandler(w http.ResponseWriter, r *http.Request) {
	// Extract name from URL path
	pathParams := r.URL.Path[len("/terminals/name/"):]
	if pathParams == "" {
		http.Error(w, "Terminal name is required", http.StatusBadRequest)
		return
	}

	name := pathParams

	terminal, err := h.service.GetTerminalByName(r.Context(), name)
	if err != nil {
		switch {
		case errors.Is(err, ErrTerminalNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, "Internal server error: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(terminal)
}

// ListTerminalsHandler handles GET /terminals
func (h *TerminalHandler) ListTerminalsHandler(w http.ResponseWriter, r *http.Request) {
	terminals, err := h.service.ListTerminals(r.Context())
	if err != nil {
		http.Error(w, "Internal server error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(terminals)
}

// UpdateTerminalHandler handles PUT /terminals/{id}
func (h *TerminalHandler) UpdateTerminalHandler(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	pathParams := r.URL.Path[len("/terminals/"):]
	if pathParams == "" {
		http.Error(w, "Terminal ID is required", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(pathParams)
	if err != nil {
		http.Error(w, "Invalid terminal ID format", http.StatusBadRequest)
		return
	}

	var terminal Terminal
	if err := json.NewDecoder(r.Body).Decode(&terminal); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Ensure the ID from URL matches the ID in request body (if provided)
	terminal.ID = id

	// Validate required fields
	if terminal.Name == "" {
		http.Error(w, "Terminal name is required", http.StatusBadRequest)
		return
	}

	err = h.service.UpdateTerminal(r.Context(), terminal)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmptyName):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrNameTooLong):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrTerminalNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, ErrDuplicateTerminalName):
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			http.Error(w, "Internal server error: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Terminal updated successfully"})
}

// DeleteTerminalHandler handles DELETE /terminals/{id}
func (h *TerminalHandler) DeleteTerminalHandler(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	pathParams := r.URL.Path[len("/terminals/"):]
	if pathParams == "" {
		http.Error(w, "Terminal ID is required", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(pathParams)
	if err != nil {
		http.Error(w, "Invalid terminal ID format", http.StatusBadRequest)
		return
	}

	err = h.service.DeleteTerminal(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, ErrTerminalNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, "Internal server error: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Terminal deleted successfully"})
}

// FindTerminalsNearHandler handles GET /terminals/near?lat=&lon=&radius=
func (h *TerminalHandler) FindTerminalsNearHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	latStr := r.URL.Query().Get("lat")
	lonStr := r.URL.Query().Get("lon")
	radiusStr := r.URL.Query().Get("radius")

	if latStr == "" || lonStr == "" || radiusStr == "" {
		http.Error(w, "Latitude, longitude, and radius parameters are required", http.StatusBadRequest)
		return
	}

	latitude, errLat := strconv.ParseFloat(latStr, 64)
	longitude, errLon := strconv.ParseFloat(lonStr, 64)
	radiusMeters, errRadius := strconv.ParseFloat(radiusStr, 64)

	if errLat != nil || errLon != nil || errRadius != nil {
		http.Error(w, "Invalid numeric parameters", http.StatusBadRequest)
		return
	}

	terminals, err := h.service.FindTerminalsNear(r.Context(), latitude, longitude, radiusMeters)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidRadius):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrInvalidCoordinates):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "Internal server error: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(terminals)
}