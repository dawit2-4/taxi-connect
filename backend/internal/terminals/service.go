package terminals

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/google/uuid"
)

// TerminalService provides business logic for terminal operations
type TerminalService struct {
	repo TerminalRepository
}

// NewTerminalService creates a new TerminalService with the given repository
func NewTerminalService(repo TerminalRepository) *TerminalService {
	return &TerminalService{repo: repo}
}

// Custom error types for the service layer
var (
	ErrTerminalNotFound      = errors.New("terminal not found")
	ErrDuplicateTerminalName = errors.New("terminal with this name already exists")
	ErrInvalidLocation       = errors.New("invalid location format")
	ErrUnexpectedNoRows      = errors.New("unexpected no rows returned")
	ErrEmptyName             = errors.New("terminal has no name")
	ErrNameTooLong           = errors.New("name cannot exceed 100 characters")
	ErrInvalidRadius         = errors.New("radius must be positive")
	ErrInvalidCoordinates    = errors.New("invalid coordinates: latitude must be [-90,90], longitude [-180,180]")
	ErrSameTerminals         = errors.New("origin and destination terminals cannot be the same")
)

// CreateTerminal validates input and creates a new terminal
func (s *TerminalService) CreateTerminal(ctx context.Context, terminal Terminal) (uuid.UUID, error) {
	// Business logic: Validate terminal name
	if terminal.Name == "" {
		return uuid.Nil, ErrEmptyName
	}
	if len(terminal.Name) > 100 {
		return uuid.Nil, ErrNameTooLong
	}
	// Business logic: Could add more validation here (e.g., name format, reserved words)
	// Example: if strings.Contains(strings.ToUpper(terminal.Name), "TEST") {
	//     return uuid.Nil, errors.New("terminal names cannot contain 'TEST'")
	// }

	// Delegate to repository
	id, err := s.repo.Create(ctx, terminal)
	if err != nil {
		// Add context to repository errors
		switch {
		case errors.Is(err, ErrDuplicateTerminalName):
			return uuid.Nil, fmt.Errorf("failed to create terminal: %w", err)
		case errors.Is(err, ErrInvalidLocation):
			return uuid.Nil, fmt.Errorf("failed to create terminal: %w", err)
		default:
			return uuid.Nil, fmt.Errorf("failed to create terminal: %w", err)
		}
	}

	return id, nil
}

// GetTerminal retrieves a terminal by its ID
func (s *TerminalService) GetTerminal(ctx context.Context, id uuid.UUID) (*Terminal, error) {
	terminal, err := s.repo.GetByID(ctx, id)
	if err != nil {
		// Preserve specific errors like NotFound for HTTP layer to handle
		switch {
		case errors.Is(err, ErrTerminalNotFound):
			return nil, ErrTerminalNotFound
		default:
			return nil, fmt.Errorf("failed to get terminal: %w", err)
		}
	}

	return terminal, nil
}

// GetTerminalByName retrieves a terminal by its name
func (s *TerminalService) GetTerminalByName(ctx context.Context, name string) (*Terminal, error) {
	if name == "" {
		return nil, errors.New("terminal name cannot be empty")
	}

	terminal, err := s.repo.GetByName(ctx, name)
	if err != nil {
		switch {
		case errors.Is(err, ErrTerminalNotFound):
			return nil, ErrTerminalNotFound
		default:
			return nil, fmt.Errorf("failed to get terminal by name: %w", err)
		}
	}

	return terminal, nil
}

// ListTerminals returns all terminals
func (s *TerminalService) ListTerminals(ctx context.Context) ([]Terminal, error) {
	terminals, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list terminals: %w", err)
	}
	return terminals, nil
}

// UpdateTerminal validates input and updates an existing terminal
func (s *TerminalService) UpdateTerminal(ctx context.Context, terminal Terminal) error {
	// Business logic: Validate terminal
	if terminal.Name == "" {
		return ErrEmptyName
	}
	if len(terminal.Name) > 100 {
		return ErrNameTooLong
	}

	// Delegate to repository
	err := s.repo.Update(ctx, terminal)
	if err != nil {
		switch {
		case errors.Is(err, ErrTerminalNotFound):
			return ErrTerminalNotFound
		case errors.Is(err, ErrDuplicateTerminalName):
			return fmt.Errorf("failed to update terminal: %w", err)
		default:
			return fmt.Errorf("failed to update terminal: %w", err)
		}
	}

	return nil
}

// DeleteTerminal removes a terminal by ID
func (s *TerminalService) DeleteTerminal(ctx context.Context, id uuid.UUID) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrTerminalNotFound):
			return ErrTerminalNotFound
		default:
			return fmt.Errorf("failed to delete terminal: %w", err)
		}
	}
	return nil
}

// FindTerminalsNear returns terminals within a specified distance (meters) of a point
func (s *TerminalService) FindTerminalsNear(ctx context.Context, latitude, longitude, radiusMeters float64) ([]Terminal, error) {
	// Business logic: Validate inputs (could also be done in repository)
	if radiusMeters <= 0 {
		return nil, ErrInvalidRadius
	}
	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		return nil, ErrInvalidCoordinates
	}

	// Delegate to repository
	terminals, err := s.repo.FindNearby(ctx, latitude, longitude, radiusMeters)
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby terminals: %w", err)
	}
	return terminals, nil
}

// QueueEstimateFunc is a callback for retrieving queue wait-time estimates
// for a terminal pair. Defined here to avoid an import cycle with the queue
// package; it is wired up in main.go using the queue service's GetEstimate.
type QueueEstimateFunc func(fromID, toID uuid.UUID) (waitMinutes int, confidence float32, err error)

// ErrNoRouteFound is returned when no path exists between two terminals.
var ErrNoRouteFound = errors.New("no route found between terminals")

// findShortestPath runs Dijkstra over the adjacency list and returns the
// shortest path (by total travel time) from fromID to toID, or nil if none exists.
func findShortestPath(edges []RouteEdge, fromID, toID uuid.UUID) []RouteEdge {
	// Build adjacency list: from_terminal_id -> list of outgoing edges
	adj := make(map[uuid.UUID][]RouteEdge)
	for _, e := range edges {
		adj[e.FromTerminalID] = append(adj[e.FromTerminalID], e)
	}

	// Dijkstra
	dist := make(map[uuid.UUID]int)
	prev := make(map[uuid.UUID]RouteEdge)
	visited := make(map[uuid.UUID]bool)

	for _, e := range edges {
		dist[e.FromTerminalID] = -1
		dist[e.ToTerminalID] = -1
	}
	dist[fromID] = 0

	for {
		// Pick unvisited node with smallest distance
		var u uuid.UUID
		best := -1
		found := false
		for node, d := range dist {
			if !visited[node] && d >= 0 && (best == -1 || d < best) {
				best = d
				u = node
				found = true
			}
		}
		if !found || u == toID {
			break
		}
		visited[u] = true

		for _, e := range adj[u] {
			if visited[e.ToTerminalID] {
				continue
			}
			nd := dist[u] + e.TravelTimeMinutes
			if dist[e.ToTerminalID] == -1 || nd < dist[e.ToTerminalID] {
				dist[e.ToTerminalID] = nd
				prev[e.ToTerminalID] = e
			}
		}
	}

	if dist[toID] == -1 {
		return nil
	}

	// Reconstruct path
	var path []RouteEdge
	for cur := toID; cur != fromID; {
		e, ok := prev[cur]
		if !ok {
			return nil
		}
		path = append(path, e)
		cur = e.FromTerminalID
	}
	// Reverse
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

// edgeKey returns a comparable key for a route edge.
func edgeKey(e RouteEdge) string {
	return e.FromTerminalID.String() + ":" + e.ToTerminalID.String()
}

// pathKey returns a comparable key for a path (sequence of edge keys).
func pathKey(path []RouteEdge) string {
	k := ""
	for _, e := range path {
		k += edgeKey(e) + ";"
	}
	return k
}

// FindAlternativeRoutes finds up to maxRoutes alternative paths from fromID to toID,
// ordered by total travel time. Each segment is enriched with queue estimates via
// getQueueEstimate (may be nil to skip queue enrichment).
func (s *TerminalService) FindAlternativeRoutes(
	ctx context.Context,
	fromID, toID uuid.UUID,
	maxRoutes int,
	getQueueEstimate QueueEstimateFunc,
) ([]AlternativeRoute, error) {
	if fromID == toID {
		return nil, ErrSameTerminals
	}

	edges, err := s.repo.GetRouteEdges(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get route edges: %w", err)
	}

	if len(edges) == 0 {
		return nil, ErrNoRouteFound
	}

	// Validate that both terminals exist
	if _, err := s.repo.GetByID(ctx, fromID); err != nil {
		if errors.Is(err, ErrTerminalNotFound) {
			return nil, fmt.Errorf("origin terminal not found: %w", ErrTerminalNotFound)
		}
		return nil, fmt.Errorf("failed to validate origin terminal: %w", err)
	}
	if _, err := s.repo.GetByID(ctx, toID); err != nil {
		if errors.Is(err, ErrTerminalNotFound) {
			return nil, fmt.Errorf("destination terminal not found: %w", ErrTerminalNotFound)
		}
		return nil, fmt.Errorf("failed to validate destination terminal: %w", err)
	}

	// Find the shortest path first
	shortest := findShortestPath(edges, fromID, toID)
	if shortest == nil {
		return nil, ErrNoRouteFound
	}

	// Collect candidate paths using a simplified Yen's k-shortest paths:
	// for each edge in the shortest path, remove it and re-run Dijkstra.
	type candidate struct {
		path []RouteEdge
		key  string
	}
	seen := make(map[string]bool)
	var candidates []candidate

	// Add the shortest path
	sk := pathKey(shortest)
	seen[sk] = true
	candidates = append(candidates, candidate{path: shortest, key: sk})

	// For each edge in the shortest path, find an alternative that avoids it
	for i := 0; i < len(shortest); i++ {
		removed := make([]RouteEdge, 0, len(edges)-1)
		for _, e := range edges {
			if edgeKey(e) == edgeKey(shortest[i]) {
				continue
			}
			removed = append(removed, e)
		}
		alt := findShortestPath(removed, fromID, toID)
		if alt != nil {
			ak := pathKey(alt)
			if !seen[ak] {
				seen[ak] = true
				candidates = append(candidates, candidate{path: alt, key: ak})
			}
		}
	}

	// Build AlternativeRoute for each candidate, compute totals
	type scoredRoute struct {
		route AlternativeRoute
		total int
	}
	var scored []scoredRoute

	for _, c := range candidates {
		var segs []RouteSegment
		totalTravel := 0
		totalWait := 0

		for _, e := range c.path {
			seg := RouteSegment{
				FromTerminalID:       e.FromTerminalID,
				ToTerminalID:         e.ToTerminalID,
				TravelTimeMinutes:    e.TravelTimeMinutes,
				EstimatedWaitMinutes: 0,
				QueueConfidence:      0,
			}
			totalTravel += e.TravelTimeMinutes

			if getQueueEstimate != nil {
				wait, conf, err := getQueueEstimate(e.FromTerminalID, e.ToTerminalID)
				if err == nil {
					seg.EstimatedWaitMinutes = wait
					seg.QueueConfidence = conf
					totalWait += wait
				}
			}
			segs = append(segs, seg)
		}

		route := AlternativeRoute{
			RouteID:                 c.key,
			TotalTravelTimeMinutes:  totalTravel,
			TotalEstimatedWaitMinutes: totalWait,
			TotalTimeMinutes:        totalTravel + totalWait,
			Segments:                segs,
			HopCount:                len(segs),
		}
		scored = append(scored, scoredRoute{route: route, total: route.TotalTimeMinutes})
	}

	// Sort by total time (travel + wait) ascending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].total < scored[j].total
	})

	// Limit to maxRoutes
	if maxRoutes <= 0 {
		maxRoutes = 3
	}
	if len(scored) > maxRoutes {
		scored = scored[:maxRoutes]
	}

	result := make([]AlternativeRoute, len(scored))
	for i, s := range scored {
		result[i] = s.route
	}
	return result, nil
}