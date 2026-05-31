package netscanner

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// ConnectionQuery holds filter, sort, and pagination parameters for listing connections.
type ConnectionQuery struct {
	Status  string // filter: granted, denied, error, informational (empty = all)
	Service string // filter: sshd, http, ... (empty = all)
	Sort    string // field: last, first, count, client, server, service (default: last)
	Order   string // asc, desc (default: desc)
	Limit   int    // 1-200 (default: 50)
	Cursor  string // opaque cursor for pagination (the ID of the last item on the previous page)
}

// ConnectionPage is a paginated response of connection infos.
type ConnectionPage struct {
	Items      []*ConnectionInfo `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
	HasMore    bool              `json:"has_more"`
	Total      int               `json:"total"`
}

// ListConnections returns a filtered, sorted, and paginated page of connections.
func (s *Server) ListConnections(ctx context.Context, q ConnectionQuery) (*ConnectionPage, error) {
	// Load all connections from the store
	all, err := s.store.SelectConnectionsByCursor(ctx)
	if err != nil {
		return nil, err
	}

	// Convert and filter
	infos := make([]*ConnectionInfo, 0, len(all))
	for _, conn := range all {
		ci := &ConnectionInfo{
			ID:      conn.ID,
			Server:  *s.deviceToDeviceInfo(ctx, conn.Server),
			Client:  *s.deviceToDeviceInfo(ctx, conn.Client),
			Service: conn.Service,
			Status:  string(conn.Status),
			Count:   conn.Count,
			First:   conn.First,
			Last:    conn.Last,
		}

		// Apply filters
		if q.Status != "" && ci.Status != q.Status {
			continue
		}
		if q.Service != "" && !strings.EqualFold(ci.Service, q.Service) {
			continue
		}
		infos = append(infos, ci)
	}

	// Sort
	sortField := q.Sort
	if sortField == "" {
		sortField = "last"
	}
	desc := q.Order != "asc"

	sort.SliceStable(infos, func(i, j int) bool {
		less := false
		switch sortField {
		case "first":
			less = infos[i].First < infos[j].First
		case "count":
			less = infos[i].Count < infos[j].Count
		case "client":
			less = (infos[i].Client.DNS + infos[i].Client.Address) < (infos[j].Client.DNS + infos[j].Client.Address)
		case "server":
			less = (infos[i].Server.DNS + infos[i].Server.Address) < (infos[j].Server.DNS + infos[j].Server.Address)
		case "service":
			less = infos[i].Service < infos[j].Service
		default: // "last"
			less = infos[i].Last < infos[j].Last
		}
		if desc {
			return !less
		}
		return less
	})

	total := len(infos)

	// Default limit
	limit := q.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	// Apply cursor pagination
	startIndex := 0
	if q.Cursor != "" {
		for i, ci := range infos {
			if ci.ID == q.Cursor {
				startIndex = i + 1 // start AFTER the cursor
				break
			}
		}
		if startIndex >= len(infos) {
			return &ConnectionPage{Items: []*ConnectionInfo{}, HasMore: false, Total: total}, nil
		}
	}

	endIndex := startIndex + limit
	if endIndex > len(infos) {
		endIndex = len(infos)
	}
	page := infos[startIndex:endIndex]

	hasMore := endIndex < len(infos)
	nextCursor := ""
	if hasMore && len(page) > 0 {
		nextCursor = page[len(page)-1].ID
	}

	return &ConnectionPage{
		Items:      page,
		NextCursor: nextCursor,
		HasMore:    hasMore,
		Total:      total,
	}, nil
}

// parseConnectionQuery extracts query parameters from an HTTP request into a ConnectionQuery.
func parseConnectionQuery(r *http.Request) ConnectionQuery {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	return ConnectionQuery{
		Status:  q.Get("status"),
		Service: q.Get("service"),
		Sort:    q.Get("sort"),
		Order:   q.Get("order"),
		Limit:   limit,
		Cursor:  q.Get("cursor"),
	}
}
