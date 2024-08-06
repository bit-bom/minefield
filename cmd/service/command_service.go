package service

import (
	"context"
	"github.com/bit-bom/minefield/pkg"
	"github.com/bit-bom/minefield/pkg/ingest"
	pb "github.com/bit-bom/minefield/proto"
)

type CommandService struct {
	pb.UnimplementedCommandServiceServer
}

func (s *CommandService) IngestSBOM(ctx context.Context, req *pb.IngestSBOMRequest) (*pb.IngestSBOMResponse, error) {
	storage := pkg.GetStorageInstance("localhost:6379")
	err := ingest.SBOM(req.SbomPath, storage)
	if err != nil {
		return nil, err
	}
	return &pb.IngestSBOMResponse{Message: "SBOM ingested successfully"}, nil
}

func (s *CommandService) Cache(ctx context.Context, req *pb.CacheRequest) (*pb.CacheResponse, error) {
	storage := pkg.GetStorageInstance("localhost:6379")
	err := pkg.Cache(storage)
	if err != nil {
		return nil, err
	}
	return &pb.CacheResponse{Message: "Cache completed successfully"}, nil
}
