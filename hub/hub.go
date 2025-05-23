package hub

import (
	"sync"

	ppvpb "github.com/yasushisakai/ppv-service/gen/go/ppv/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Hub struct {
	mu sync.RWMutex
	// key is jobid
	subs     map[string][]chan *ppvpb.ComputeStatus
	active   map[string]struct{}
	finished map[string]*ppvpb.ComputeStatus
}

func New() *Hub {
	return &Hub{
		mu:       sync.RWMutex{},
		subs:     make(map[string][]chan *ppvpb.ComputeStatus),
		active:   make(map[string]struct{}),
		finished: make(map[string]*ppvpb.ComputeStatus),
	}
}

func (h *Hub) Declare(jobID string) {
	h.mu.Lock()
	h.active[jobID] = struct{}{}
	defer h.mu.Unlock()
}

func (h *Hub) Register(jobID string) (<-chan *ppvpb.ComputeStatus, func(), error) {

	h.mu.RLock()

	// if finished status, return it

	if fin, ok := h.finished[jobID]; ok {
		h.mu.RUnlock()
		ch := make(chan *ppvpb.ComputeStatus, 1)
		ch <- fin
		close(ch)
		return ch, func() {}, nil
	}

	// if not active status, return error
	if _, ok := h.active[jobID]; !ok {
		h.mu.RUnlock()
		return nil, nil, status.Error(codes.NotFound, "unknown job")
	}

	h.mu.RUnlock()

	// if not wait
	ch := make(chan *ppvpb.ComputeStatus, 4)

	h.mu.Lock()
	h.subs[jobID] = append(h.subs[jobID], ch)
	h.mu.Unlock()

	// client uses this to unregister itself
	// in the future
	unregister := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		list := h.subs[jobID]
		// look for the channel
		for i, c := range h.subs[jobID] {
			if c == ch {
				// close the ch
				h.subs[jobID] = append(list[:i], list[i+i:]...)
				break
			}
		}
		close(ch)
	}

	return ch, unregister, nil
}

func (h *Hub) Broadcast(jobID string, status *ppvpb.ComputeStatus) {

	h.mu.RLock()
	list := append([]chan *ppvpb.ComputeStatus(nil), h.subs[jobID]...)
	h.mu.RUnlock()

	// send it to subs
	for _, ch := range list {
		select {
		case ch <- status:
		default:
		}
	}

	if status.Status == ppvpb.ComputeStatus_FINISHED {
		h.mu.Lock()
		h.finished[jobID] = status
		delete(h.active, jobID)
		for _, ch := range h.subs[jobID] {
			close(ch)
		}
		delete(h.subs, jobID)
		h.mu.Unlock()
	}

}

func (h *Hub) ListJobs() []string {

	list := make([]string, 0, len(h.active)+len(h.finished))

	for k := range h.active {
		list = append(list, k)
	}

	for k := range h.finished {
		list = append(list, k)
	}

	return list
}
