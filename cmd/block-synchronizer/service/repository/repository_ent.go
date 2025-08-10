package repository

import (
	"context"
	"fmt"

	"gno.land-block-indexer/ent"
	"gno.land-block-indexer/ent/block"
	"gno.land-block-indexer/lib/log"
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
func (r *repositoryBsEnt) GetNotSequentialBlockNum(ctx context.Context) (int, error) {
	// Check start with 0 and find the block stopping at the first non-sequential block
	// 1, 2 ,3, 6 => 3 is not sequential

	const batchSize = 1000
	currentHeight := 1

	for {
		// Check if this range has the expected number of blocks
		endHeight := currentHeight + batchSize - 1
		count, err := r.client.Block.
			Query().
			Where(block.HeightGTE(currentHeight), block.HeightLTE(endHeight)).
			Count(ctx)
		if err != nil {
			r.logger.Errorf("failed to count blocks from %d to %d: %v", currentHeight, endHeight, err)
			return 0, fmt.Errorf("failed to count blocks: %w", err)
		}

		// If count matches expected, all blocks exist - move to next batch
		if count == batchSize {
			currentHeight += batchSize
			continue
		}

		// Count doesn't match - there's a gap in this range
		// Get blocks from currentHeight to endHeight and find the first missing one
		blocks, err := r.client.Block.
			Query().
			Where(block.HeightGTE(currentHeight), block.HeightLTE(endHeight)).
			Order(ent.Asc("height")).
			Select("height").
			All(ctx)
		if err != nil {
			r.logger.Errorf("failed to query blocks from %d to %d: %v", currentHeight, endHeight, err)
			return 0, fmt.Errorf("failed to query blocks: %w", err)
		}

		// Find the first gap in this specific range
		expectedHeight := currentHeight
		for _, block := range blocks {
			if block.Height != expectedHeight {
				return expectedHeight, nil
			}
			expectedHeight++
		}

		// If we processed all blocks in range but still missing some,
		// the gap is right after the last processed block
		return expectedHeight, nil
	}
}
