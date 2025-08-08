package repository

import (
	"context"

	"gno.land-block-indexer/model"
)

type Repository interface {
	// block operations
	AddBlock(ctx context.Context, block *model.Block) error
	AddBlocks(ctx context.Context, blocks []*model.Block) error
	GetBlock(ctx context.Context, blockNum int) (*model.Block, error)
	GetBlocks(ctx context.Context, offset int, limit int) ([]*model.Block, error)
	GetHighestBlock(ctx context.Context) (*model.Block, error)

	// transaction operations
	AddTransaction(ctx context.Context, blockNum int, tx *model.Transaction) error
	AddTransactions(ctx context.Context, blockNum int, txs []*model.Transaction) error
	GetTransaction(ctx context.Context, txHash string) (*model.Transaction, error)
	GetTransactions(ctx context.Context, blockNum int, offset int, limit int) ([]*model.Transaction, error)
}
