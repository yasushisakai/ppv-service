package main

import (
	"context"
	"log"
	"net"
	"os/signal"
	"syscall"

	ppvpb "github.com/yasushisakai/ppv-service/gen/go/ppv/v1"
	"github.com/yasushisakai/ppv-service/hub"
	"github.com/yasushisakai/ppv-service/queue"
	"github.com/yasushisakai/ppv-service/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {

	log.Println("starting ppv server...")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT)
	defer stop()

	log.Println("starting hub...")
	hub := hub.New()

	queue := queue.New(100, hub)

	log.Println("starting queue...")

	queue.Start(ctx, 0)

	log.Println("queue started")

	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(600*1024*1024), // 600MB
		grpc.MaxSendMsgSize(600*1024*1024), // 600MB
	)
	reflection.Register(grpcServer)
	ppvpb.RegisterPPVServiceServer(grpcServer, &server.ComputeServer{Q: queue, H: hub})

	log.Printf("starting server...")

	port := ":50051"
	listener, err := net.Listen("tcp", port)

	if err != nil {
		log.Fatalf("error listening port %s: %v", port, err)
	}

	log.Printf("listening %s", port)

	go func() {
		<-ctx.Done()
		log.Println("shutting down")
		grpcServer.GracefulStop()
	}()

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("failed to serve grpcServer %s: %v", port, err)
	}

}
