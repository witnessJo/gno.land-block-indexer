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
	repo := repository.NewRepositoryEnt(logger, config.EntConfig)

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

	// Create worker pool for async processing
	workCh := make(chan msgbroker.BlockWithTransactions, 100)
	const numWorkers = 5

	// Start workers
	for i := 0; i < numWorkers; i++ {
		go s.messageWorker(ctx, workCh, i)
	}

	// Subscribe to the topic
	err := s.msgBroker.Subscribe(topicBlockWithTransactions, func(message []byte) error {
		s.logger.Debugf("Received message of size: %d bytes", len(message))

		// Unmarshal the message into BlockWithTransactions struct
		var blockWithTxs msgbroker.BlockWithTransactions
		err := json.Unmarshal(message, &blockWithTxs)
		if err != nil {
			return s.logger.Errorf("Failed to unmarshal message: %v", err)
		}

		// Send to worker pool (non-blocking)
		select {
		case workCh <- blockWithTxs:
			// Message sent to worker successfully
			return nil
		default:
			// Worker pool is full, log and drop message
			s.logger.Warnf("Worker pool full, dropping block: Height=%d", blockWithTxs.Block.Height)
			return nil
		}
	})

	// Check for errors in subscription
	if err != nil {
		return s.logger.Errorf("Failed to subscribe to topic %s: %v", topicBlockWithTransactions, err)
	}
	s.logger.Infof("Subscribed to topic %s successfully", topicBlockWithTransactions)

	// Keep running until context is cancelled
	<-ctx.Done()
	s.logger.Infof("Context done, stopping subscription")
	close(workCh)
	return ctx.Err()
}

// messageWorker processes messages in a worker pool pattern
func (s *service) messageWorker(ctx context.Context, workCh <-chan msgbroker.BlockWithTransactions, workerID int) {
	s.logger.Infof("Starting message worker %d", workerID)
	for {
		select {
		case <-ctx.Done():
			s.logger.Infof("Message worker %d stopping", workerID)
			return
		case blockWithTxs, ok := <-workCh:
			if !ok {
				s.logger.Infof("Message worker %d: work channel closed", workerID)
				return
			}

			// Process message
			if err := s.processBlockWithTransactions(ctx, blockWithTxs); err != nil {
				s.logger.Errorf("Worker %d failed to process block %d: %v",
					workerID, blockWithTxs.Block.Height, err)
			}
		}
	}
}

// processBlockWithTransactions handles the actual message processing
func (s *service) processBlockWithTransactions(ctx context.Context, blockWithTxs msgbroker.BlockWithTransactions) error {
	err := s.repo.AddBlock(ctx, blockWithTxs.Block)
	if err != nil {
		return s.logger.Errorf("failed to add block %d: %w", blockWithTxs.Block.Height, err)
	}

	err = s.repo.AddTransactions(ctx, blockWithTxs.Block.Height, blockWithTxs.Transactions)
	if err != nil {
		return s.logger.Errorf("failed to add transactions for block %d: %w", blockWithTxs.Block.Height, err)
	}

	s.logger.Infof("Successfully processed block %d with %d transactions",
		blockWithTxs.Block.Height, len(blockWithTxs.Transactions))
	return nil
}
