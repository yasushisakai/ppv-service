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

	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

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
			return nil, nil, 0, false, err
		}

		switch st.Status {
		case ppvpb.ComputeStatus_FINISHED:
			log.Println("Job Finished")
			return st.Consensus, st.Influence, st.Iteration, st.DidConverge, nil
		}
	}
}
