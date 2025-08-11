package repository

import (
	"context"
	"fmt"

	"gno.land-block-indexer/ent"
	"gno.land-block-indexer/ent/block"
	"gno.land-block-indexer/lib/log"

	_ "github.com/lib/pq" // PostgreSQL driver
)

type repositoryBsEnt struct {
	logger log.Logger
	client *ent.Client
}

func NewRepositoryBsEnt(logger log.Logger, config *RepositoryBsEntConfig) RepositoryBs {
	client, err := ent.Open("postgres", "host="+config.Host+
		" port="+fmt.Sprintf("%d", config.Port)+
		" user="+config.User+
		" password="+config.Password+
		" dbname="+config.Database+
		" sslmode=disable")
	if err != nil {
		panic("failed to connect to database: " + err.Error())
	}

	// create the schema if it doesn't exist
	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	return &repositoryBsEnt{
		logger: logger,
		client: client,
	}
}

// GetNotSequentialBlockNum implements BsRepository.
func (r *repositoryBsEnt) GetNotSequentialBlockNum(ctx context.Context, limit int) (int, error) {
	// Check start with 0 and find the block stopping at the first non-sequential block
	// 1, 2 ,3, 6 => 3 is not sequential

	const batchSize = 1000
	found := false
	startBlockHeight := 1
	for ; startBlockHeight < limit; startBlockHeight += batchSize {
		// Check if this range has the expected number of blocks
		endHeight := startBlockHeight + batchSize - 1
		count, err := r.client.Block.
			Query().
			Where(block.IDGTE(startBlockHeight), block.IDLTE(endHeight)).
			Count(ctx)
		if err != nil {
			r.logger.Errorf("failed to count blocks from %d to %d: %v", startBlockHeight, endHeight, err)
			return 0, fmt.Errorf("failed to count blocks: %w", err)
		}
		if count < batchSize {
			r.logger.Infof("found non-sequential block at height %d, expected %d blocks but got %d", startBlockHeight, batchSize, count)
			found = true
			break
		}
		r.logger.Debugf("found %d blocks from %d to %d", count, startBlockHeight, endHeight)
		startBlockHeight += batchSize
	}

	if !found {
		r.logger.Infof("all blocks are sequential up to height %d", startBlockHeight-1)
		return limit, nil
	}

	// We need find exactly the first non-sequential block
	candidateBlocks, err := r.client.Block.Query().
		Where(block.IDGTE(startBlockHeight)).
		Order(ent.Asc(block.FieldID)).
		Limit(batchSize).
		All(ctx)
	if err != nil {
		r.logger.Errorf("failed to query blocks starting from %d: %v", startBlockHeight, err)
		return 0, fmt.Errorf("failed to query blocks: %w", err)
	}
	if len(candidateBlocks) == 0 {
		r.logger.Infof("no blocks found starting from height %d", startBlockHeight)
		return limit, nil
	}

	for _, b := range candidateBlocks {
		if b.ID != startBlockHeight {
			r.logger.Infof("found non-sequential block at height %d, expected %d", b.ID, startBlockHeight)
			return startBlockHeight - 1, nil
		}
		startBlockHeight++
	}

	return limit, nil
}
