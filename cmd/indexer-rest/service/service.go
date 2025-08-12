package service

import (
	"context"

	"gno.land-block-indexer/lib/log"
	"gno.land-block-indexer/model"
	"gno.land-block-indexer/repository"
)

type Service interface {
	GetTokenBalances(ctx context.Context, address string) ([]model.TokenBalance, error)
	GetTokenAccountBalances(ctx context.Context, tokenPath string, address string) ([]model.Account, error)
	GetTransferHistory(ctx context.Context, address string) ([]model.Transfer, error)
}

type service struct {
	logger log.Logger
	repo   repository.Repository
}

func NewService(logger log.Logger, repo repository.Repository) Service {
	return &service{
		logger: logger,
		repo:   repo,
	}
}

// GetAccountBalances implements Service.
func (s *service) GetTokenBalances(ctx context.Context, address string) ([]model.TokenBalance, error) {
	tokenBalances, err := s.repo.GetBalances(ctx, address)
	if err != nil {
		return nil, s.logger.Errorf("Failed to get account balances for address %s: %v", address, err)
	}
	return tokenBalances, nil
}

// GetTokenBalances implements Service.
func (s *service) GetTokenAccountBalances(ctx context.Context, token, address string) ([]model.Account, error) {
	accounts, err := s.repo.GetAccounts(ctx, address, token)
	if err != nil {
		return nil, s.logger.Errorf("Failed to get token balances for address %s and token %s: %v", address, token, err)
	}

	return accounts, nil
}

// GetTransferHistory implements Service.
func (s *service) GetTransferHistory(ctx context.Context, address string) ([]model.Transfer, error) {
	transfers, err := s.repo.GetTransfers(ctx, address, address, "")
	if err != nil {
		s.logger.Errorf("Failed to get transfer history for address %s: %v", address, err)
		return nil, err
	}

	return transfers, nil
}
