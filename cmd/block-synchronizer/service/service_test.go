package service

import (
	"context"
	"testing"
	"time"

	// "time"
	"gno.land-block-indexer/externals/msgbroker"
	"gno.land-block-indexer/lib/log"
	"gno.land-block-indexer/model"
	"gno.land-block-indexer/repository"
)

func GetTestService(ctx context.Context) Service {
	logger := log.NewLogger()
	config := &ServiceConfig{
		EntConfig: &repository.RepositoryEntConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "postgres",
			Database: "postgres",
		},
		FetchEndpoint:     "https://indexer.onbloc.xyz/graphql/query",
		WebSocketEndpoint: "wss://indexer.onbloc.xyz/graphql/query",
		LocalStackConfig: &msgbroker.LocalStackConfig{
			Endpoint: "http://localhost:4566", // LocalStack endpoint
			Region:   "us-east-1",             // LocalStack region
		},
	}
	svc := NewService(ctx, logger, config)
	if svc == nil {
		panic("Failed to create service")
	}
	return svc
}

func TestPollBlocks(t *testing.T) {
	ctx := context.Background()
	service := GetTestService(ctx)
	blocks, err := service.PollBlocks(0, 100000)
	if err != nil {
		t.Fatalf("Failed to poll blocks: %v", err)
	}
	if len(blocks) == 0 {
		t.Fatal("Expected to retrieve blocks, got none")
	}
	t.Logf("Retrieved %d blocks", len(blocks))

	for _, block := range blocks {
		if block.Height <= 0 {
			t.Errorf("Expected block height to be greater than 0, got %d", block.Height)
		}
		if block.Hash == "" {
			t.Error("Expected block hash to be non-empty")
		}
	}
}

func TestPollTransactions(t *testing.T) {
	ctx := context.Background()
	service := GetTestService(ctx)
	transactions, err := service.PollTransactions(0, 100000)
	if err != nil {
		t.Fatalf("Failed to poll transactions: %v", err)
	}
	if len(transactions) == 0 {
		t.Fatal("Expected to retrieve transactions, got none")
	}
	t.Logf("Retrieved %d transactions", len(transactions))

	for _, tx := range transactions {
		if tx.Hash == "" {
			t.Error("Expected transaction hash to be non-empty")
		}
		if tx.BlockHeight <= 0 {
			t.Errorf("Expected transaction block height to be greater than 0, got %d", tx.BlockHeight)
		}
	}
}

func TestSubscribeToBlocks(t *testing.T) {
	ctx := context.Background()
	ch := make(chan model.Block, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // 30초 동안 실행
	defer cancel()

	service := GetTestService(ctx)

	go func() {
		err := service.SubscribeLastestBlock(ctx, ch)
		if err != nil && err != context.DeadlineExceeded {
			t.Errorf("SubscribeToBlocks error: %v", err)
		}
	}()

	blockCount := 0
	for {
		select {
		case block := <-ch:
			blockCount++
			t.Logf("Block #%d received: height=%d, hash=%s",
				blockCount, block.Height, block.Hash)

			if blockCount >= 3 { // 3개 블록 받으면 성공
				t.Logf("Successfully received %d blocks", blockCount)
				return
			}

		case <-ctx.Done():
			if blockCount > 0 {
				t.Logf("Test completed: received %d blocks", blockCount)
			} else {
				t.Error("No blocks received within timeout")
			}
			return
		}
	}
}
