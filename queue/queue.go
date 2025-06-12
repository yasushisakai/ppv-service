package queue

import (
	"context"
	"log"
	"runtime"

	"github.com/google/uuid"
	"github.com/yasushisakai/ppv"
	ppvpb "github.com/yasushisakai/ppv-service/gen/go/ppv/v1"

	"github.com/yasushisakai/ppv-service/hub"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Job struct {
	ID      string
	Request interface{} // Can be *ppvpb.ComputeRequest or *ppvpb.DotRequest
}

type Queue struct {
	jobs chan Job
	hub  *hub.Hub
}

func New(capacity int, h *hub.Hub) *Queue {
	return &Queue{
		jobs: make(chan Job, capacity),
		hub:  h,
	}
}

func (q *Queue) Enqueue(ctx context.Context, req interface{}) (string, error) {
	jobID := uuid.NewString()
	q.hub.Declare(jobID)
	job := Job{
		ID:      jobID,
		Request: req,
	}

	select {
	// happy path
	case q.jobs <- job:
		// Emit QUEUED status based on request type
		switch req.(type) {
		case *ppvpb.ComputeRequest:
			status := ppvpb.ComputeStatus{Status: ppvpb.ComputeStatus_QUEUED}
			q.hub.Broadcast(jobID, &status)
		case *ppvpb.DotRequest:
			dotStatus := ppvpb.DotStatus{Status: ppvpb.DotStatus_QUEUED}
			q.hub.BroadcastDot(jobID, &dotStatus)
		}
		return jobID, nil
	default:
		// fall through
	}

	if _, ok := ctx.Deadline(); !ok {
		return "", status.Error(codes.ResourceExhausted, "queue is full")
	}

	select {
	// try again
	case q.jobs <- job:
		// Emit QUEUED status based on request type
		switch req.(type) {
		case *ppvpb.ComputeRequest:
			status := ppvpb.ComputeStatus{Status: ppvpb.ComputeStatus_QUEUED}
			q.hub.Broadcast(jobID, &status)
		case *ppvpb.DotRequest:
			dotStatus := ppvpb.DotStatus{Status: ppvpb.DotStatus_QUEUED}
			q.hub.BroadcastDot(jobID, &dotStatus)
		}
		return jobID, nil
	case <-ctx.Done():
		return "", status.Error(codes.DeadlineExceeded, "queue still full")
	}
}

func (q *Queue) Start(ctx context.Context, n int) {

	if n <= 0 {
		n = runtime.NumCPU()
	}

	log.Printf("Starting %d queue workers", n)

	for i := 0; i < n; i++ {
		go q.worker(ctx)
	}

}

func (q *Queue) worker(ctx context.Context) {
	log.Println("Worker started")
	for {
		select {
		case <-ctx.Done():
			log.Println("Worker stopping due to context cancellation")
			return
		case job := <-q.jobs:
			log.Printf("Worker processing job %s", job.ID)

			switch req := job.Request.(type) {
			case *ppvpb.ComputeRequest:
				status := ppvpb.ComputeStatus{Status: ppvpb.ComputeStatus_PROCESSING}
				q.hub.Broadcast(job.ID, &status)

				consensus, influence, iteration, didConverge, err := ppv.Compute(
					req.Matrix,
					int(req.NDelegate),
					int(req.NPolicy),
					int(req.NInterm),
				)

				if err != nil {
					status = ppvpb.ComputeStatus{
						Status:       ppvpb.ComputeStatus_ERROR,
						ErrorMessage: err.Error(),
					}
				} else {
					status = ppvpb.ComputeStatus{
						Status:      ppvpb.ComputeStatus_FINISHED,
						Consensus:   consensus,
						Influence:   influence,
						Iteration:   int32(iteration),
						DidConverge: didConverge,
					}
				}

				q.hub.Broadcast(job.ID, &status)

			case *ppvpb.DotRequest:
				dotStatus := ppvpb.DotStatus{Status: ppvpb.DotStatus_PROCESSING}
				q.hub.BroadcastDot(job.ID, &dotStatus)

				output, err := ppv.Dot(req.A, req.B, int(req.Size))

				if err != nil {
					dotStatus = ppvpb.DotStatus{
						Status:       ppvpb.DotStatus_ERROR,
						ErrorMessage: err.Error(),
					}
				} else {
					dotStatus = ppvpb.DotStatus{
						Status: ppvpb.DotStatus_FINISHED,
						Output: output,
					}
				}

				q.hub.BroadcastDot(job.ID, &dotStatus)

			default:
				log.Printf("Unknown job type for job %s", job.ID)
			}
		}
	}
}
