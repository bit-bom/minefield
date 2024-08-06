package service

import (
	"context"
	"github.com/bit-bom/minefield/pkg"
	pb "github.com/bit-bom/minefield/proto"
)

type QueryService struct {
	pb.UnimplementedQueryServiceServer
}

func (s *QueryService) QueryDependencies(ctx context.Context, req *pb.QueryRequest) (*pb.QueryResponse, error) {
	storage := pkg.GetStorageInstance("localhost:6379")
	result, err := pkg.QueryDependencies(req.QueryString, storage)
	if err != nil {
		return nil, err
	}
	dependencies := []*pb.Dependency{}
	for _, dep := range result {
		dependencies = append(dependencies, &pb.Dependency{
			Name: dep.Name,
			Type: dep.Type,
			Id:   dep.ID,
		})
	}
	return &pb.QueryResponse{Dependencies: dependencies}, nil
}

func (s *QueryService) GenerateLeaderboard(ctx context.Context, req *pb.LeaderboardRequest) (*pb.LeaderboardResponse, error) {
	storage := pkg.GetStorageInstance("localhost:6379")
	result, err := pkg.GenerateLeaderboard(req.Script, storage)
	if err != nil {
		return nil, err
	}
	entries := []*pb.LeaderboardEntry{}
	for _, entry := range result {
		entries = append(entries, &pb.LeaderboardEntry{
			Name:        entry.Name,
			Type:        entry.Type,
			QueryLength: entry.QueryLength,
		})
	}
	return &pb.LeaderboardResponse{Entries: entries}, nil
}
