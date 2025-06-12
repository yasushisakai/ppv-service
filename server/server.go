package server

import (
	"context"
	"io"
	"log"

	ppvpb "github.com/yasushisakai/ppv-service/gen/go/ppv/v1"
	"github.com/yasushisakai/ppv-service/hub"
	"github.com/yasushisakai/ppv-service/queue"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ComputeServer struct {
	ppvpb.UnimplementedPPVServiceServer
	Q *queue.Queue
	H *hub.Hub
}

func (s *ComputeServer) RequestCompute(ctx context.Context, req *ppvpb.ComputeRequest) (*ppvpb.ComputeResponse, error) {

	jobID, err := s.Q.Enqueue(ctx, req)

	if err != nil {
		return nil, err
	}

	log.Printf("job %s enqueued", jobID)

	return &ppvpb.ComputeResponse{JobId: jobID}, nil
}

func (s *ComputeServer) WaitCompute(req *ppvpb.WaitRequest, stream ppvpb.PPVService_WaitComputeServer) error {

	ch, done, err := s.H.Register(req.JobId)

	if err != nil {
		return err
	}

	defer done()

	for {
		select {

		case st, ok := <-ch:
			if !ok {
				log.Printf("channel closed for job %s", req.JobId)
				return nil
			}

			// if something, send it to the stream
			if err := stream.Send(st); err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}

		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

func (s *ComputeServer) ListJobs(context.Context, *emptypb.Empty) (*ppvpb.JobList, error) {
	ids := s.H.ListJobs()
	jobList := &ppvpb.JobList{
		Ids: ids,
	}

	return jobList, nil
}

func (s *ComputeServer) RequestDot(ctx context.Context, req *ppvpb.DotRequest) (*ppvpb.DotResponse, error) {

	jobID, err := s.Q.Enqueue(ctx, req)

	if err != nil {
		return nil, err
	}

	log.Printf("dot job %s enqueued", jobID)

	return &ppvpb.DotResponse{JobId: jobID}, nil
}

func (s *ComputeServer) WaitDot(req *ppvpb.WaitRequest, stream ppvpb.PPVService_WaitDotServer) error {

	ch, done, err := s.H.RegisterDot(req.JobId)

	if err != nil {
		return err
	}

	defer done()

	for {
		select {

		case st, ok := <-ch:
			if !ok {
				log.Printf("channel closed for dot job %s", req.JobId)
				return nil
			}

			// if something, send it to the stream
			if err := stream.Send(st); err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}

		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}
