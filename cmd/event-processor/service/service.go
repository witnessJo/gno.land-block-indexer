package service

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"gno.land-block-indexer/externals/msgbroker"
	"gno.land-block-indexer/lib/log"
	"gno.land-block-indexer/model"
	"gno.land-block-indexer/repository"
)

const (
	TOPIC_BLOCK_WITH_TXS = "block_with_txs"
	UNIT_NAME            = "ugnot" // The unit name for the token, can be changed as needed
)

type Service interface {
	// usecase (from controller)
	SubscribeAndHandle(ctx context.Context) error

	//
	ProcessBlockWithTransactions(ctx context.Context, blockWithTxs msgbroker.BlockWithTransactions) error
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
	err := s.msgBroker.Subscribe(TOPIC_BLOCK_WITH_TXS, func(message []byte) error {
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
		return s.logger.Errorf("Failed to subscribe to topic %s: %v", TOPIC_BLOCK_WITH_TXS, err)
	}
	s.logger.Infof("Subscribed to topic %s successfully", TOPIC_BLOCK_WITH_TXS)

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
			if err := s.ProcessBlockWithTransactions(ctx, blockWithTxs); err != nil {
				s.logger.Errorf("Worker %d failed to process block %d: %v",
					workerID, blockWithTxs.Block.Height, err)
			}
		}
	}
}

// processBlockWithTransactions handles the actual message processing
func (s *service) ProcessBlockWithTransactions(ctx context.Context, blockWithTxs msgbroker.BlockWithTransactions) error {
	exists, err := s.repo.AddBlock(ctx, blockWithTxs.Block)
	if err != nil {
		return s.logger.Errorf("failed to add block %d: %w", blockWithTxs.Block.Height, err)
	}
	if exists {
		s.logger.Debugf("Block %d already exists, skipping", blockWithTxs.Block.Height)
		return nil
	}

	err = s.repo.AddTransactions(ctx, blockWithTxs.Block.Height, blockWithTxs.Transactions)
	if err != nil {
		return s.logger.Errorf("failed to add transactions for block %d: %w", blockWithTxs.Block.Height, err)
	}

	s.logger.Infof("Successfully processed block %d with %d transactions", blockWithTxs.Block.Height, len(blockWithTxs.Transactions))

	// Parse transactions to extract transfers and account updates
	if err := s.parseAndProcessTransactions(ctx, blockWithTxs.Transactions); err != nil {
		return s.logger.Errorf("failed to parse and process transactions for block %d: %w", blockWithTxs.Block.Height, err)
	}

	return nil
}

// parseAndProcessTransactions parses transactions to extract transfers and account information
func (s *service) parseAndProcessTransactions(ctx context.Context, transactions []model.Transaction) error {
	var err error

	for _, tx := range transactions {
		transfers := make([]model.Transfer, 0)
		for _, event := range tx.Response.Events {
			if strings.ToLower(event.Type) == "transfer" {
				var fromAddress, toAddress string
				var numValue int64

				for _, attr := range event.Attrs {
					if attr.Key == "from" {
						fromAddress = attr.Value
					} else if attr.Key == "to" {
						toAddress = attr.Value
					} else if attr.Key == "value" {
						numValue, err = strconv.ParseInt(attr.Value, 10, 64)
						if err != nil {
							return s.logger.Errorf("Invalid 'value' attribute in transfer event for transaction %s: %v", tx.Hash, err)
						}
					}
				}

				switch strings.ToLower(event.Func) {
				case "mint":
					if err := s.handleMintEvent(ctx, &tx, event.PkgPath, toAddress, numValue); err != nil {
						return s.logger.Errorf("Failed to handle mint event for transaction %s: %v", tx.Hash, err)
					}
				case "burn":
					if err := s.handleBurnEvent(ctx, &tx, event.PkgPath, fromAddress, int64(numValue)); err != nil {
						return s.logger.Errorf("Failed to handle burn event for transaction %s: %v", tx.Hash, err)
					}
				case "transfer":
					if err := s.handleTransferEvent(ctx, &tx, event.PkgPath, fromAddress, toAddress, int64(numValue)); err != nil {
						return s.logger.Errorf("Failed to handle transfer event for transaction %s: %v", tx.Hash, err)
					}
				default:
					s.logger.Warnf("Unknown event type %s in transaction %s", event.Type, tx.Hash)
				}

				transfers = append(transfers, model.Transfer{
					FromAddress: fromAddress,
					ToAddress:   toAddress,
					Token:       event.PkgPath,
					CreatedAt:   time.Now(),
					Amount:      float64(numValue),
					Denom:       UNIT_NAME,
					Func:        event.Func,
				})
			}
		}

		s.logger.Debugf("😀 Transfer count for transaction %s: %d", tx.Hash, len(transfers))
		err := s.repo.AddTransfers(ctx, &tx, transfers)
		if err != nil {
			return s.logger.Errorf("Failed to add transfers for transaction %s: %v", tx.Hash, err)
		}
	}

	return nil
}

func (s *service) handleMintEvent(ctx context.Context, tx *model.Transaction, pkg string, toAddress string, value int64) error {
	// For mint events, tokens are created and added to the toAddress
	// Check if account exists, create if not
	if toAddress == "" {
		s.logger.Warnf("Transfer event for transaction %s has empty 'to' address, skipping", tx.Hash)
		return nil
	}

	toAccount, err := s.repo.GetAccount(ctx, toAddress, pkg)
	if err != nil {
		return err
	}
	if toAccount == nil {
		err := s.repo.AddAccount(ctx, &model.Account{
			Address:   toAddress,
			Token:     pkg,
			Amount:    0, // Initialize with zero balance
			CreatedAt: time.Now(),
		})
		if err != nil {
			return s.logger.Errorf("Failed to add account %s: %v", toAddress, err)
		}
	}

	// Increment the balance for the minted tokens
	if err := s.repo.IncrementAccountBalance(ctx, toAddress, pkg, value); err != nil {
		return s.logger.Errorf("Failed to increment balance for account %s: %v", toAddress, err)
	}

	return nil
}

func (s *service) handleBurnEvent(ctx context.Context, tx *model.Transaction, pkg string, fromAddress string, value int64) error {
	// For burn events, tokens are destroyed from the fromAddress
	// Check if account exists
	if fromAddress == "" {
		s.logger.Warnf("Transfer event for transaction %s has empty 'from' address, skipping", tx.Hash)
		return nil
	}

	fromAccount, err := s.repo.GetAccount(ctx, fromAddress, pkg)
	if err != nil {
		return err
	}
	if fromAccount == nil {
		// If account doesn't exist, create it with zero balance (shouldn't happen in normal burn scenario)
		err := s.repo.AddAccount(ctx, &model.Account{
			Address:   fromAddress,
			Token:     pkg,
			Amount:    0,
			CreatedAt: time.Now(),
		})
		if err != nil {
			return s.logger.Errorf("Failed to add account %s: %v", fromAddress, err)
		}
	}

	// Decrement the balance for the burned tokens (negative value for burn)
	if err := s.repo.IncrementAccountBalance(ctx, fromAddress, pkg, -value); err != nil {
		return s.logger.Errorf("Failed to decrement balance for account %s: %v", fromAddress, err)
	}

	return nil
}

func (s *service) handleTransferEvent(ctx context.Context, tx *model.Transaction, pkg string, fromAddress string, toAddress string, value int64) error {
	fromAccount, err := s.repo.GetAccount(ctx, fromAddress, pkg)
	if err != nil {
		return err
	}
	if fromAccount == nil {
		err := s.repo.AddAccount(ctx, &model.Account{
			Address: fromAddress,
			Token:   pkg,
			Amount:  0, // Initialize with zero balance
		})
		if err != nil {
			return s.logger.Errorf("Failed to add account %s: %v", fromAddress, err)
		}
	}

	toAccount, err := s.repo.GetAccount(ctx, toAddress, pkg)
	if err != nil {
		return err
	}
	if toAccount == nil {
		err := s.repo.AddAccount(ctx, &model.Account{
			Address: toAddress,
			Token:   pkg,
			Amount:  0, // Initialize with zero balance
		})
		if err != nil {
			return s.logger.Errorf("Failed to add account %s: %v", toAddress, err)
		}
	}

	if err := s.repo.IncrementAccountBalance(ctx, fromAddress, pkg, -(value)); err != nil {
		return s.logger.Errorf("Failed to decrement balance for account %s: %v", fromAddress, err)
	}
	if err := s.repo.IncrementAccountBalance(ctx, toAddress, pkg, value); err != nil {
		return s.logger.Errorf("Failed to increment balance for account %s: %v", toAddress, err)
	}

	return nil
}
