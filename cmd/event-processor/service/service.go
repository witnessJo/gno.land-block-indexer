package service

import (
	"context"
	"encoding/json"

	"gno.land-block-indexer/externals/msgbroker"
	"gno.land-block-indexer/lib/log"
	"gno.land-block-indexer/repository"
)

const (
	topicBlockWithTransactions = "block_with_txs"
)

type Service interface {
	// usecase (from controller)
	SubscribeAndHandle(ctx context.Context) error
}

type service struct {
	logger    log.Logger
	repo      repository.Repository
	msgBroker msgbroker.MsgBroker
}

type ServiceConfig struct {
	EntConfig        *repository.RepositoryEntConfig
	LocalStackConfig *msgbroker.LocalStackConfig
}

func NewService(ctx context.Context, logger log.Logger, config *ServiceConfig) Service {
	repo := repository.NewRepositoryEnt(config.EntConfig)

	localStack, err := msgbroker.NewMsgBrokerLocalStack(ctx, logger, config.LocalStackConfig)
	if err != nil {
		logger.Fatalf("Failed to create local stack message broker", "error", err)
	}

	return &service{
		logger:    logger,
		repo:      repo,
		msgBroker: localStack,
	}
}

// SubscribeAndHandle implements Service.
func (s *service) SubscribeAndHandle(ctx context.Context) error {
	s.logger.Infof("Starting subscription to block with transactions topic")

	// Subscribe to the topic
	err := s.msgBroker.Subscribe(topicBlockWithTransactions, func(message []byte) error {
		s.logger.Infof("Received message: %s", message)
		// Unmarshal the message into BlockWithTransactions struct
		var blockWithTxs msgbroker.BlockWithTransactions
		err := json.Unmarshal(message, &blockWithTxs)
		if err != nil {
			return s.logger.Errorf("Failed to unmarshal message: %v", err)
		}

		err = s.repo.AddBlock(ctx, blockWithTxs.Block)
		if err != nil {
			return s.logger.Errorf("Failed to add block: %v", err)
		}

		err = s.repo.AddTransactions(ctx, blockWithTxs.Block.Height, blockWithTxs.Transactions)
		if err != nil {
			return s.logger.Errorf("Failed to add transactions: %v", err)
		}
		s.logger.Infof("Successfully added block and transactions to repository")
		return nil
	})

	// Check for errors in subscription
	if err != nil {
		return s.logger.Errorf("Failed to subscribe to topic %s: %v", topicBlockWithTransactions, err)
	}
	s.logger.Infof("Subscribed to topic %s successfully", topicBlockWithTransactions)

	// Wait for a while before subscribing again
	select {
	case <-ctx.Done():
		s.logger.Infof("Context done, stopping subscription")
		return nil
	default:
		s.logger.Infof("Waiting for new messages...")
	}
	return nil
}
