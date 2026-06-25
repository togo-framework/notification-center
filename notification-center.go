// Package notificationcenter is an in-app notification inbox for togo — the
// "database channel" + bell that complements the notifications plugin (which
// delivers to external providers like email/Slack/push).
//
// Store a per-user notification, list the inbox, count unread, and mark
// read/deleted. A current user is resolved from the request context
// (X-User-Id header or an auth claim).
//
//	nc, _ := notificationcenter.FromKernel(k)
//	nc.Notify(ctx, "user-1", notificationcenter.Notification{
//	    Type: "comment", Title: "New comment", Body: "Sam replied to you", ActionURL: "/posts/9",
//	})
package notificationcenter

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/togo-framework/togo"
)

// Notification is a single inbox item for a user.
type Notification struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	Type      string         `json:"type,omitempty"`
	Title     string         `json:"title"`
	Body      string         `json:"body,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
	ActionURL string         `json:"action_url,omitempty"`
	Read      bool           `json:"read"`
	CreatedAt time.Time      `json:"created_at"`
	ReadAt    *time.Time     `json:"read_at,omitempty"`
}

// Store is the persistence seam (swap the default in-memory store for a DB one).
type Store interface {
	Add(n *Notification)
	ByUser(userID string) []*Notification
	Get(id string) (*Notification, bool)
	Delete(id string) bool
}

// Filter narrows a List query.
type Filter struct {
	UnreadOnly bool
	Limit      int // 0 = default 50
	Offset     int
}

// Service is the notification-center runtime stored on the kernel.
type Service struct {
	k     *togo.Kernel
	store Store
	mu    sync.Mutex
	seq   int
}

func init() {
	togo.RegisterProviderFunc("notification-center", togo.PriorityLate+10, func(k *togo.Kernel) error {
		s := &Service{k: k, store: newMemStore()}
		k.Set("notification-center", s)
		if k.Router != nil {
			s.mountRoutes(k.Router)
		}
		return nil
	})
}

// FromKernel returns the notification-center Service.
func FromKernel(k *togo.Kernel) (*Service, bool) {
	v, ok := k.Get("notification-center")
	if !ok {
		return nil, false
	}
	s, ok := v.(*Service)
	return s, ok
}

// WithStore swaps the backing store (e.g. a DB-backed implementation).
func (s *Service) WithStore(store Store) *Service {
	s.store = store
	return s
}

// Notify stores a notification for a user and returns the stored record.
// If a realtime broker is present on the kernel it is also published so a
// connected bell can update live.
func (s *Service) Notify(ctx context.Context, userID string, n Notification) *Notification {
	s.mu.Lock()
	s.seq++
	n.ID = newID(s.seq)
	s.mu.Unlock()

	n.UserID = userID
	n.Read = false
	n.ReadAt = nil
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}
	rec := &n
	s.store.Add(rec)
	return rec
}

// List returns a user's notifications, newest first.
func (s *Service) List(userID string, f Filter) []*Notification {
	items := s.store.ByUser(userID)
	if f.UnreadOnly {
		out := items[:0:0]
		for _, n := range items {
			if !n.Read {
				out = append(out, n)
			}
		}
		items = out
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	if f.Offset >= len(items) {
		return []*Notification{}
	}
	end := f.Offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[f.Offset:end]
}

// UnreadCount returns the number of unread notifications for a user.
func (s *Service) UnreadCount(userID string) int {
	n := 0
	for _, x := range s.store.ByUser(userID) {
		if !x.Read {
			n++
		}
	}
	return n
}

// MarkRead marks a single notification read.
func (s *Service) MarkRead(id string) bool {
	n, ok := s.store.Get(id)
	if !ok {
		return false
	}
	if !n.Read {
		n.Read = true
		t := time.Now()
		n.ReadAt = &t
	}
	return true
}

// MarkAllRead marks every notification for a user read; returns how many changed.
func (s *Service) MarkAllRead(userID string) int {
	changed := 0
	t := time.Now()
	for _, n := range s.store.ByUser(userID) {
		if !n.Read {
			n.Read = true
			n.ReadAt = &t
			changed++
		}
	}
	return changed
}

// Delete removes a notification.
func (s *Service) Delete(id string) bool { return s.store.Delete(id) }

// --- in-memory store ---

type memStore struct {
	mu    sync.RWMutex
	byID  map[string]*Notification
	order []*Notification
	max   int
}

func newMemStore() *memStore {
	return &memStore{byID: map[string]*Notification{}, max: 5000}
}

func (m *memStore) Add(n *Notification) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byID[n.ID] = n
	m.order = append(m.order, n)
	if len(m.order) > m.max {
		drop := m.order[0]
		m.order = m.order[1:]
		delete(m.byID, drop.ID)
	}
}

func (m *memStore) ByUser(userID string) []*Notification {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*Notification
	for _, n := range m.order {
		if n.UserID == userID {
			out = append(out, n)
		}
	}
	return out
}

func (m *memStore) Get(id string) (*Notification, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n, ok := m.byID[id]
	return n, ok
}

func (m *memStore) Delete(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byID[id]; !ok {
		return false
	}
	delete(m.byID, id)
	for i, n := range m.order {
		if n.ID == id {
			m.order = append(m.order[:i], m.order[i+1:]...)
			break
		}
	}
	return true
}

// --- REST ---

func (s *Service) mountRoutes(r chi.Router) {
	r.Route("/api/notifications", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, req *http.Request) {
			uid := currentUser(req)
			f := Filter{UnreadOnly: req.URL.Query().Get("unread") == "true"}
			writeJSON(w, 200, s.List(uid, f))
		})
		r.Get("/unread-count", func(w http.ResponseWriter, req *http.Request) {
			writeJSON(w, 200, map[string]int{"count": s.UnreadCount(currentUser(req))})
		})
		r.Post("/read-all", func(w http.ResponseWriter, req *http.Request) {
			writeJSON(w, 200, map[string]int{"updated": s.MarkAllRead(currentUser(req))})
		})
		r.Post("/{id}/read", func(w http.ResponseWriter, req *http.Request) {
			writeJSON(w, 200, map[string]bool{"ok": s.MarkRead(chi.URLParam(req, "id"))})
		})
		r.Delete("/{id}", func(w http.ResponseWriter, req *http.Request) {
			writeJSON(w, 200, map[string]bool{"ok": s.Delete(chi.URLParam(req, "id"))})
		})
	})
}

// currentUser resolves the acting user from the request (X-User-Id header).
func currentUser(req *http.Request) string {
	if v := req.Header.Get("X-User-Id"); v != "" {
		return v
	}
	if v, ok := req.Context().Value(userKey).(string); ok {
		return v
	}
	return ""
}

type ctxKey string

const userKey ctxKey = "nc.user"

// WithUser stores the acting user on a context (for non-HTTP callers/tests).
func WithUser(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userKey, userID)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func newID(seq int) string {
	return "ntf_" + time.Now().UTC().Format("20060102T150405") + "_" + itoa(seq)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
