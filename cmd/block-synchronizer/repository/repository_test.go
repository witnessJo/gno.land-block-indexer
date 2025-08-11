package repository

import (
	"context"
	"testing"

	"gno.land-block-indexer/lib/log"
)

func GetTestRepository(ctx context.Context) RepositoryBs {
	logger := log.NewLogger()
	return NewRepositoryBsEnt(logger, &RepositoryBsEntConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		Database: "postgres",
	})
}

func TestGetNotSequentialBlockNum(t *testing.T) {
	ctx := context.Background()
	repo := GetTestRepository(ctx)
	if repo == nil {
		t.Fatal("Failed to create repository")
	}
	blockNum, err := repo.GetNotSequentialBlockNum(ctx, 1000000)
	if err != nil {
		t.Fatalf("Failed to get not sequential block number: %v", err)
	}
	t.Logf("Not sequential block number: %d", blockNum)
}
