package repository

import (
	"context"
	"fmt"

	"gno.land-block-indexer/ent"
	"gno.land-block-indexer/ent/restorehistory"
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
	// Get the restore history to determine the starting point
	history, err := r.client.RestoreHistory.Query().
		Where(restorehistory.IDEQ(1)).
		First(ctx)
	if ent.IsNotFound(err) {
		r.logger.Infof("No restoring history found, returning default block number 1")
		return 1, nil
	} else if err != nil {
		return 0, r.logger.Errorf("Error querying restoring history: %v", err)
	}

	return history.RestoreRangeStart, nil
}

// SaveRestoringHistory implements RepositoryBs.
func (r *repositoryBsEnt) SaveRestoringHistory(ctx context.Context, startBlock int, endBlock int) error {
	r.logger.Infof("Saving restoring history from block %d to %d", startBlock, endBlock)

	// Save only one entry for the same range (id:1)
	err := r.client.RestoreHistory.Create().
		SetID(1).
		SetRestoreRangeStart(startBlock).
		SetRestoreRangeEnd(endBlock).
		SetBeingBlock(0).
		OnConflictColumns(restorehistory.FieldID).
		UpdateRestoreRangeStart().
		UpdateRestoreRangeEnd().
		UpdateBeingBlock().
		Exec(ctx)
	if err != nil {
		r.logger.Errorf("failed to save restoring history from %d to %d: %v", startBlock, endBlock, err)
		return fmt.Errorf("failed to save restoring history: %w", err)
	}

	return nil
}
