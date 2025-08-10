package controller

import (
	"context"
	"gno.land-block-indexer/cmd/block-synchronizer/service"
	"gno.land-block-indexer/externals/msgbroker"
	"gno.land-block-indexer/lib/log"
	"gno.land-block-indexer/repository"
)

type Controller struct {
	service service.Service
}

type ControllerConfig struct {
	repoConfig *repository.RepositoryEntConfig
}

func NewController() *Controller {
	ctx := context.Background()
	logger := log.NewLogger()
	service := service.NewService(ctx, logger, &service.ServiceConfig{
		FetchEndpoint:     "https://indexer.onbloc.xyz/graphql/query",
		WebSocketEndpoint: "wss://indexer.onbloc.xyz/graphql/query",
		EntConfig: &repository.RepositoryEntConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "postgres",
			Database: "postgres",
		},
		LocalStackConfig: &msgbroker.LocalStackConfig{
			Endpoint: "http://localhost:4566",
			Region:   "us-east-1",
		},
	})

	return &Controller{
		service: service,
	}
}

func (c *Controller) Run(ctx context.Context) error {
	// go c.service.SubscribeAndPush(ctx)
	go c.service.RestoreMissingBlockAndTransactions(ctx)
	return nil
}
