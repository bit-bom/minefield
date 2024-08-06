package main

import (
	"github.com/bit-bom/minefield/cmd/service"
	pb "github.com/bit-bom/minefield/proto"
	"google.golang.org/grpc"
	"net"
	"net/http"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		panic(err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterCommandServiceServer(grpcServer, &service.CommandService{})
	pb.RegisterQueryServiceServer(grpcServer, &service.QueryService{})

	mux := http.NewServeMux()
	mux.Handle(pb.NewCommandServiceHandler(&service.CommandService{}))
	mux.Handle(pb.NewQueryServiceHandler(&service.QueryService{}))

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: handler.New(mux),
	}

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			panic(err)
		}
	}()

	if err := httpServer.ListenAndServe(); err != nil {
		panic(err)
	}
}
