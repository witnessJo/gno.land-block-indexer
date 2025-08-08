package repository

import (
	"context"
	"fmt"
	"log"
	"time"

	"gno.land-block-indexer/ent"
	"gno.land-block-indexer/ent/block"
	"gno.land-block-indexer/ent/schema"
	"gno.land-block-indexer/ent/transaction"
	"gno.land-block-indexer/model"

	_ "github.com/lib/pq"
)

type RepositoryEntConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

func NewRepositoryEnt(config *RepositoryEntConfig) Repository {
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

	return &RepositoryEnt{
		client: client,
	}
}

type RepositoryEnt struct {
	client *ent.Client
}

// GetHighestBlock implements Repository.
func (r *RepositoryEnt) GetHighestBlock(ctx context.Context) (*model.Block, error) {
	entBlock, err := r.client.Block.Query().
		Order(ent.Desc("height")).
		First(ctx)
	if err != nil {
		log.Printf("failed to get highest block: %v", err)
		return nil, err
	}
	return &model.Block{
		Hash:     entBlock.Hash,
		Height:   entBlock.Height,
		Time:     entBlock.Time,
		TotalTxs: entBlock.TotalTxs,
		NumTxs:   entBlock.NumTxs,
	}, nil
}

// AddBlock implements Repository.
func (r *RepositoryEnt) AddBlock(ctx context.Context, block *model.Block) error {
	// Convert model.Block to ent.Block
	// Use the ent client to create the block in the database
	_, err := r.client.Block.Create().
		SetHash(block.Hash).
		SetHeight(block.Height).
		SetTime(block.Time).
		SetTotalTxs(block.TotalTxs).
		SetNumTxs(block.NumTxs).
		SetCreatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		log.Printf("failed to add block: %v", err)
		return err
	}

	return nil
}

// AddBlocks implements Repository.
func (r *RepositoryEnt) AddBlocks(ctx context.Context, blocks []*model.Block) error {
	if len(blocks) == 0 {
		return nil
	}

	bulk := make([]*ent.BlockCreate, len(blocks))
	for i, block := range blocks {
		bulk[i] = r.client.Block.Create().
			SetHash(block.Hash).
			SetHeight(block.Height).
			SetTime(block.Time).
			SetTotalTxs(block.TotalTxs).
			SetNumTxs(block.NumTxs).
			SetCreatedAt(time.Now())
	}

	_, err := r.client.Block.CreateBulk(bulk...).Save(ctx)
	if err != nil {
		log.Printf("failed to add blocks: %v", err)
		return err
	}

	return nil
}

// AddTransaction implements Repository.
func (r *RepositoryEnt) AddTransaction(ctx context.Context, blockNum int, tx *model.Transaction) error {
	_, err := r.client.Transaction.Create().
		SetIndex(tx.Index).
		SetHash(tx.Hash).
		SetSuccess(tx.Success).
		SetBlockHeight(tx.BlockHeight).
		SetGasWanted(tx.GasWanted).
		SetGasUsed(tx.GasUsed).
		SetMemo(tx.Memo).
		SetGasFee(schema.GasFee(tx.GasFee)).
		SetMessages(convertMessagesToSchema(tx.Messages)).
		SetResponse(convertResponseToSchema(tx.Response)).
		SetCreatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		log.Printf("failed to add transaction: %v", err)
		return err
	}

	return nil
}

// AddTransactions implements Repository.
func (r *RepositoryEnt) AddTransactions(ctx context.Context, blockNum int, txs []*model.Transaction) error {
	if len(txs) == 0 {
		return nil
	}

	bulk := make([]*ent.TransactionCreate, len(txs))
	for i, tx := range txs {
		bulk[i] = r.client.Transaction.Create().
			SetIndex(tx.Index).
			SetHash(tx.Hash).
			SetSuccess(tx.Success).
			SetBlockHeight(tx.BlockHeight).
			SetGasWanted(tx.GasWanted).
			SetGasUsed(tx.GasUsed).
			SetMemo(tx.Memo).
			SetGasFee(schema.GasFee(tx.GasFee)).
			SetMessages(convertMessagesToSchema(tx.Messages)).
			SetResponse(convertResponseToSchema(tx.Response)).
			SetCreatedAt(time.Now())
	}

	_, err := r.client.Transaction.CreateBulk(bulk...).Save(ctx)
	if err != nil {
		log.Printf("failed to add transactions: %v", err)
		return err
	}

	return nil
}

// GetBlock implements Repository.
func (r *RepositoryEnt) GetBlock(ctx context.Context, blockNum int) (*model.Block, error) {
	entBlock, err := r.client.Block.Query().
		Where(block.HeightEQ(blockNum)).
		Only(ctx)
	if err != nil {
		log.Printf("failed to get block: %v", err)
		return nil, err
	}

	return &model.Block{
		Hash:     entBlock.Hash,
		Height:   entBlock.Height,
		Time:     entBlock.Time,
		TotalTxs: entBlock.TotalTxs,
		NumTxs:   entBlock.NumTxs,
	}, nil
}

// GetBlocks implements Repository.
func (r *RepositoryEnt) GetBlocks(ctx context.Context, offset int, limit int) ([]*model.Block, error) {
	entBlocks, err := r.client.Block.Query().
		Offset(offset).
		Limit(limit).
		Order(ent.Desc("height")).
		All(ctx)
	if err != nil {
		log.Printf("failed to get blocks: %v", err)
		return nil, err
	}

	blocks := make([]*model.Block, len(entBlocks))
	for i, entBlock := range entBlocks {
		blocks[i] = &model.Block{
			Hash:     entBlock.Hash,
			Height:   entBlock.Height,
			Time:     entBlock.Time,
			TotalTxs: entBlock.TotalTxs,
			NumTxs:   entBlock.NumTxs,
		}
	}

	return blocks, nil
}

// GetTransaction implements Repository.
func (r *RepositoryEnt) GetTransaction(ctx context.Context, txHash string) (*model.Transaction, error) {
	entTx, err := r.client.Transaction.Query().
		Where(transaction.HashEQ(txHash)).
		Only(ctx)
	if err != nil {
		log.Printf("failed to get transaction: %v", err)
		return nil, err
	}

	return &model.Transaction{
		Index:       entTx.Index,
		Hash:        entTx.Hash,
		Success:     entTx.Success,
		BlockHeight: entTx.BlockHeight,
		GasWanted:   entTx.GasWanted,
		GasUsed:     entTx.GasUsed,
		Memo:        entTx.Memo,
		GasFee:      model.GasFee(entTx.GasFee),
		Messages:    convertSchemaMessagesToModel(entTx.Messages),
		Response:    convertSchemaResponseToModel(entTx.Response),
	}, nil
}

// GetTransactions implements Repository.
func (r *RepositoryEnt) GetTransactions(ctx context.Context, blockNum int, offset int, limit int) ([]*model.Transaction, error) {
	entTxs, err := r.client.Transaction.Query().
		Where(transaction.BlockHeightEQ(blockNum)).
		Offset(offset).
		Limit(limit).
		Order(ent.Asc("index")).
		All(ctx)
	if err != nil {
		log.Printf("failed to get transactions: %v", err)
		return nil, err
	}

	txs := make([]*model.Transaction, len(entTxs))
	for i, entTx := range entTxs {
		txs[i] = &model.Transaction{
			Index:       entTx.Index,
			Hash:        entTx.Hash,
			Success:     entTx.Success,
			BlockHeight: entTx.BlockHeight,
			GasWanted:   entTx.GasWanted,
			GasUsed:     entTx.GasUsed,
			Memo:        entTx.Memo,
			GasFee:      model.GasFee(entTx.GasFee),
			Messages:    convertSchemaMessagesToModel(entTx.Messages),
			Response:    convertSchemaResponseToModel(entTx.Response),
		}
	}

	return txs, nil
}

// Conversion functions between model and schema types
func convertMessagesToSchema(modelMessages []model.Message) []schema.Message {
	schemaMessages := make([]schema.Message, len(modelMessages))
	for i, msg := range modelMessages {
		schemaMessages[i] = schema.Message{
			Route:   msg.Route,
			TypeUrl: msg.TypeUrl,
			Value:   msg.Value,
		}
	}
	return schemaMessages
}

func convertResponseToSchema(modelResponse model.Response) schema.Response {
	return schema.Response{
		Log:    modelResponse.Log,
		Info:   modelResponse.Info,
		Error:  modelResponse.Error,
		Data:   modelResponse.Data,
		Events: modelResponse.Events,
	}
}

func convertSchemaMessagesToModel(schemaMessages []schema.Message) []model.Message {
	modelMessages := make([]model.Message, len(schemaMessages))
	for i, msg := range schemaMessages {
		modelMessages[i] = model.Message{
			Route:   msg.Route,
			TypeUrl: msg.TypeUrl,
			Value:   msg.Value,
		}
	}
	return modelMessages
}

func convertSchemaResponseToModel(schemaResponse schema.Response) model.Response {
	return model.Response{
		Log:    schemaResponse.Log,
		Info:   schemaResponse.Info,
		Error:  schemaResponse.Error,
		Data:   schemaResponse.Data,
		Events: schemaResponse.Events,
	}
}
