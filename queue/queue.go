package queue

import (
	"context"
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
	Request *ppvpb.ComputeRequest
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

func (q *Queue) Enqueue(ctx context.Context, req *ppvpb.ComputeRequest) (string, error) {
	jobID := uuid.NewString()
	q.hub.Declare(jobID)
	job := Job{
		ID:      jobID,
		Request: req,
	}

	select {
	// happy path
	case q.jobs <- job:
		status := ppvpb.ComputeStatus{Status: ppvpb.ComputeStatus_QUEUED}
		q.hub.Broadcast(jobID, &status)
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
		status := ppvpb.ComputeStatus{Status: ppvpb.ComputeStatus_QUEUED}
		q.hub.Broadcast(jobID, &status)
		return jobID, nil
	case <-ctx.Done():
		return "", status.Error(codes.DeadlineExceeded, "queue still full")
	}
}

func (q *Queue) Start(ctx context.Context, n int) {

	if n <= 0 {
		n = runtime.NumCPU()
	}

	for i := 0; i < n; i++ {
		go q.worker(ctx)
	}

}

func (q *Queue) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-q.jobs:

			status := ppvpb.ComputeStatus{Status: ppvpb.ComputeStatus_PROCESSING}
			q.hub.Broadcast(job.ID, &status)

			consensus, influence := ppv.Compute(
				job.Request.Matrix,
				int(job.Request.NDelegate),
				int(job.Request.NPolicy),
				int(job.Request.NInterm),
			)

			status = ppvpb.ComputeStatus{
				Status:    ppvpb.ComputeStatus_FINISHED,
				Consensus: consensus,
				Influence: influence,
			}
			q.hub.Broadcast(job.ID, &status)

		}
	}
}
