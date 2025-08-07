package service

import (
	"testing"

	"gno.land-block-indexer/repository"
)

func GetTestService() Service {
	config := &ServiceConfig{
		EntConfig: repository.RepositoryEntConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "postgres",
			Database: "postgres",
		},
		GraphqlEndpoint: "https://indexer.onbloc.xyz/graphql/query",
	}
	svc := NewService(config)
	if svc == nil {
		panic("Failed to create service")
	}
	return svc
}

func TestPollBlocks(t *testing.T) {
	service := GetTestService()
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
	service := GetTestService()
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
