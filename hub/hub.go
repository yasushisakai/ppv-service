package hub

import (
	"log"
	"sync"
	"time"

	ppvpb "github.com/yasushisakai/ppv-service/gen/go/ppv/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Hub struct {
	mu sync.RWMutex
	// key is jobid
	subs        map[string][]chan *ppvpb.ComputeStatus
	dotSubs     map[string][]chan *ppvpb.DotStatus
	active      map[string]struct{}
	finished    map[string]*ppvpb.ComputeStatus
	dotFinished map[string]*ppvpb.DotStatus
}

func New() *Hub {
	return &Hub{
		mu:          sync.RWMutex{},
		subs:        make(map[string][]chan *ppvpb.ComputeStatus),
		dotSubs:     make(map[string][]chan *ppvpb.DotStatus),
		active:      make(map[string]struct{}),
		finished:    make(map[string]*ppvpb.ComputeStatus),
		dotFinished: make(map[string]*ppvpb.DotStatus),
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
				h.subs[jobID] = append(list[:i], list[i+1:]...)
				break
			}
		}
	}

	return ch, unregister, nil
}

func (h *Hub) Broadcast(jobID string, status *ppvpb.ComputeStatus) {

	h.mu.RLock()
	list := append([]chan *ppvpb.ComputeStatus(nil), h.subs[jobID]...)
	h.mu.RUnlock()

	log.Printf("Broadcasting status %v to %d subscribers for job %s", status.Status, len(list), jobID)

	// send it to subs
	for _, ch := range list {
		select {
		case ch <- status:
			log.Printf("Successfully sent status to subscriber for job %s", jobID)
		default:
			log.Printf("Failed to send status to subscriber for job %s (channel full or closed)", jobID)
		}
	}

	if status.Status == ppvpb.ComputeStatus_FINISHED {
		// Store the finished status immediately for late subscribers
		h.mu.Lock()
		h.finished[jobID] = status
		delete(h.active, jobID)
		h.mu.Unlock()
		
		// Keep channels open for 30 seconds to allow late subscribers
		go func() {
			time.Sleep(30 * time.Second)
			h.mu.Lock()
			defer h.mu.Unlock()
			for _, ch := range h.subs[jobID] {
				close(ch)
			}
			delete(h.subs, jobID)
			
			// Clean up finished job after 5 minutes
			time.AfterFunc(5*time.Minute, func() {
				h.mu.Lock()
				defer h.mu.Unlock()
				delete(h.finished, jobID)
				log.Printf("Cleaned up finished job %s from cache", jobID)
			})
		}()
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

func (h *Hub) RegisterDot(jobID string) (<-chan *ppvpb.DotStatus, func(), error) {

	h.mu.RLock()

	// if finished status, return it

	if fin, ok := h.dotFinished[jobID]; ok {
		h.mu.RUnlock()
		ch := make(chan *ppvpb.DotStatus, 1)
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
	ch := make(chan *ppvpb.DotStatus, 4)

	h.mu.Lock()
	h.dotSubs[jobID] = append(h.dotSubs[jobID], ch)
	h.mu.Unlock()

	// client uses this to unregister itself
	// in the future
	unregister := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		list := h.dotSubs[jobID]
		// look for the channel
		for i, c := range h.dotSubs[jobID] {
			if c == ch {
				// close the ch
				h.dotSubs[jobID] = append(list[:i], list[i+1:]...)
				break
			}
		}
	}

	return ch, unregister, nil
}

func (h *Hub) BroadcastDot(jobID string, status *ppvpb.DotStatus) {

	h.mu.RLock()
	list := append([]chan *ppvpb.DotStatus(nil), h.dotSubs[jobID]...)
	h.mu.RUnlock()

	log.Printf("Broadcasting dot status %v to %d subscribers for job %s", status.Status, len(list), jobID)

	// send it to subs
	for _, ch := range list {
		select {
		case ch <- status:
			log.Printf("Successfully sent dot status to subscriber for job %s", jobID)
		default:
			log.Printf("Failed to send dot status to subscriber for job %s (channel full or closed)", jobID)
		}
	}

	if status.Status == ppvpb.DotStatus_FINISHED {
		// Store the finished status immediately for late subscribers
		h.mu.Lock()
		h.dotFinished[jobID] = status
		delete(h.active, jobID)
		h.mu.Unlock()
		
		// Keep channels open for 30 seconds to allow late subscribers
		go func() {
			time.Sleep(30 * time.Second)
			h.mu.Lock()
			defer h.mu.Unlock()
			for _, ch := range h.dotSubs[jobID] {
				close(ch)
			}
			delete(h.dotSubs, jobID)
			
			// Clean up finished job after 5 minutes
			time.AfterFunc(5*time.Minute, func() {
				h.mu.Lock()
				defer h.mu.Unlock()
				delete(h.dotFinished, jobID)
				log.Printf("Cleaned up finished dot job %s from cache", jobID)
			})
		}()
	}

}
