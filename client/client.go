package client

import (
	"context"
	"log"

	ppvpb "github.com/yasushisakai/ppv-service/gen/go/ppv/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ComputeClient struct {
	cl ppvpb.PPVServiceClient
}

func New(addr, port string, opts ...grpc.DialOption) (*ComputeClient, error) {

	opts = append(opts, 
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(600*1024*1024), // 600MB
			grpc.MaxCallSendMsgSize(600*1024*1024), // 600MB
		),
	)

	address := "passthrough:///" + addr + ":" + port

	conn, err := grpc.NewClient(address, opts...)

	if err != nil {
		return nil, err
	}

	return &ComputeClient{cl: ppvpb.NewPPVServiceClient(conn)}, nil
}

func (c *ComputeClient) Compute(
	ctx context.Context,
	matrix []float64,
	nDelegate, nPolicy, nInterm int) ([]float64, []float64, int32, bool, error) {

	// submit request
	res, err := c.cl.RequestCompute(ctx, &ppvpb.ComputeRequest{
		Matrix:    matrix,
		NDelegate: int32(nDelegate),
		NPolicy:   int32(nPolicy),
		NInterm:   int32(nInterm)})

	if err != nil {
		return nil, nil, 0, false, err
	}

	log.Printf("Job Submitted ID: %s", res.JobId)

	// wait request
	stream, err := c.cl.WaitCompute(ctx, &ppvpb.WaitRequest{JobId: res.JobId})

	if err != nil {
		return nil, nil, 0, false, err
	}

	// waiting
	for {
		st, err := stream.Recv()
		if err != nil {
			log.Printf("Error receiving stream: %v", err)
			return nil, nil, 0, false, err
		}

		log.Printf("Received status: %v for job %s", st.Status, res.JobId)

		switch st.Status {
		case ppvpb.ComputeStatus_QUEUED:
			log.Println("Job Queued")
		case ppvpb.ComputeStatus_PROCESSING:
			log.Println("Job Processing")
		case ppvpb.ComputeStatus_FINISHED:
			log.Println("Job Finished")
			return st.Consensus, st.Influence, st.Iteration, st.DidConverge, nil
		}
	}
}

func (c *ComputeClient) Dot(
	ctx context.Context,
	a []float64,
	b []float64,
	size int) ([]float64, error) {

	// submit request
	res, err := c.cl.RequestDot(ctx, &ppvpb.DotRequest{
		A:    a,
		B:    b,
		Size: int32(size)})

	if err != nil {
		return nil, err
	}

	log.Printf("Dot Job Submitted ID: %s", res.JobId)

	// wait request
	stream, err := c.cl.WaitDot(ctx, &ppvpb.WaitRequest{JobId: res.JobId})

	if err != nil {
		return nil, err
	}

	// waiting
	for {
		st, err := stream.Recv()
		if err != nil {
			log.Printf("Error receiving stream: %v", err)
			return nil, err
		}

		log.Printf("Received dot status: %v for job %s", st.Status, res.JobId)

		switch st.Status {
		case ppvpb.DotStatus_QUEUED:
			log.Println("Dot Job Queued")
		case ppvpb.DotStatus_PROCESSING:
			log.Println("Dot Job Processing")
		case ppvpb.DotStatus_FINISHED:
			log.Println("Dot Job Finished")
			return st.Output, nil
		}
	}
}
