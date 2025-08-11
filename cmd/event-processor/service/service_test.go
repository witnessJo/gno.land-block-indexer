package service

import (
	"context"
	"testing"
	"time"

	"gno.land-block-indexer/externals/msgbroker"
	"gno.land-block-indexer/lib/log"
	"gno.land-block-indexer/model"
	"gno.land-block-indexer/repository"
)

func GetTestService(ctx context.Context) Service {
	logger := log.NewLogger()
	mb, err := msgbroker.NewMsgBrokerLocalStack(ctx, logger, &msgbroker.LocalStackConfig{
		Endpoint: "http://localhost:4566",
		Region:   "us-east-1",
	})
	if err != nil {
		logger.Fatalf("Failed to create local stack message broker", "error", err)
	}

	return &service{
		logger: logger,
		repo: repository.NewRepositoryEnt(
			logger,
			&repository.RepositoryEntConfig{
				Database: "postgres",
				User:     "postgres",
				Password: "password",
				Host:     "localhost",
				Port:     5432,
			},
		),
		msgBroker: mb,
	}
}

func TestProcessBlockWithTransactions(t *testing.T) {
	ctx := context.Background()
	s := GetTestService(ctx)
	tcs := []msgbroker.BlockWithTransactions{
		{
			Block: &model.Block{
				Hash:     "B1F8E8C7A5F4D3C2B1A0987654321ABCDEF0123456789",
				Height:   12345,
				Time:     time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
				TotalTxs: 5,
				NumTxs:   1,
			},
			Transactions: []model.Transaction{
				{
					Index:       0,
					Hash:        "TXH1234567890ABCDEF0123456789ABCDEF012345678",
					Success:     true,
					BlockHeight: 12345,
					GasWanted:   200000,
					GasUsed:     150000,
					Memo:        "Token transfer test transaction",
					GasFee:      model.GasFee{Amount: 1000, Denom: "ugnot"},
					Messages:    []model.Message{},
					Response: model.Response{
						Log:   "Transaction executed successfully",
						Info:  "Token transfer completed",
						Error: "",
						Data:  "0x1234567890abcdef",
						Events: []model.Event{
							{
								Type:    "Transfer",
								Func:    "Mint",
								PkgPath: "gno.land/r/gnoswap/v1/test_token/bar",
								Attrs: []struct {
									Key   string "json:\"key\""
									Value string "json:\"value\""
								}{
									{
										Key:   "from",
										Value: "",
									},
									{
										Key:   "to",
										Value: "g17290cwvmrapvp869xfnhhawa8sm9edpufzat7d",
									},
									{
										Key:   "value",
										Value: "100000000000000", // 1 token in base units
									},
								},
							},
							{
								Type:    "Transfer",
								Func:    "Burn",
								PkgPath: "gno.land/r/gnoswap/v1/test_token/bar",
								Attrs: []struct {
									Key   string "json:\"key\""
									Value string "json:\"value\""
								}{
									{
										Key:   "g17290cwvmrapvp869xfnhhawa8sm9edpufzat7d",
										Value: "",
									},
									{
										Key:   "to",
										Value: "",
									},
									{
										Key:   "value",
										Value: "100000000000000", // 0.5 token in base units
									},
								},
							},
							{
								Type:    "Transfer",
								Func:    "Transfer",
								PkgPath: "gno.land/r/gnoswap/v1/test_token/bar",
								Attrs: []struct {
									Key   string "json:\"key\""
									Value string "json:\"value\""
								}{
									{
										Key:   "from",
										Value: "g16a7etgm9z2r653ucl36rj0l2yqcxgrz2jyegzx",
									},
									{
										Key:   "to",
										Value: "g17290cwvmrapvp869xfnhhawa8sm9edpufzat7d",
									},
									{
										Key:   "value",
										Value: "100000000000000", // 0.5 token in base units
									},
								},
							},
						},
					},
				},
			},
		},
	}

	err := s.ProcessBlockWithTransactions(ctx, tcs[0])
	if err != nil {
		t.Fatalf("Failed to process block with transactions: %v", err)
	}
}
