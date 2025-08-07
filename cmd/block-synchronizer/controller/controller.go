package controller

import (
	"context"

	"github.com/graphql-go/graphql"
	"gno.land-block-indexer/cmd/block-synchronizer/service"
	"gno.land-block-indexer/model"
	"gno.land-block-indexer/repository"
)

type Controller struct {
	schema  *graphql.Schema
	service service.Service
}

type ControllerConfig struct {
	Schema     *graphql.Schema
	repoConfig *repository.RepositoryEntConfig
}

func NewController() *Controller {
	return &Controller{}
}

func (c *Controller) Run(ctx context.Context) error {
	// TODO: Implement controller logic
	// check missing blocks and transactions
	go func() {
		for {
			ch := make(chan model.Block)
			err := c.service.SubscribeToBlocks(ctx, ch)
			if err != nil {
				// Handle error (e.g., log it, retry, etc.)
				return
			}
			select {
			case block := <-ch:
				// Process the block received from the subscription
				err = c.service.PushBlock(&block)
				if err != nil {
					// Handle error (e.g., log it, retry, etc.)
					return
				}
				transactions, err := c.service.PollTransactions(block.Height, 1)
				if err != nil {
					// Handle error (e.g., log it, retry, etc.)
					return
				}

				for _, tx := range transactions {
					err = c.service.PushTransaction(&tx)
					if err != nil {
						// Handle error (e.g., log it, retry, etc.)
						return
					}
				}

			case <-ctx.Done():
				// Context cancelled, exit the loop
				return
			}
		}
	}()

	return nil
}
