package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gno.land-block-indexer/ent"
	"gno.land-block-indexer/ent/account"
	"gno.land-block-indexer/ent/block"
	"gno.land-block-indexer/ent/schema"
	"gno.land-block-indexer/ent/transaction"
	"gno.land-block-indexer/ent/transfer"
	"gno.land-block-indexer/model"

	_ "github.com/lib/pq"
	"gno.land-block-indexer/lib/log"
)

type RepositoryEntConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

func NewRepositoryEnt(logger log.Logger, config *RepositoryEntConfig) Repository {
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
		logger.Fatalf("failed creating schema resources: %v", err)
	}

	client = client.Debug() // Enable debug mode for development

	return &RepositoryEnt{
		logger: logger,
		client: client,
	}
}

type RepositoryEnt struct {
	logger log.Logger
	client *ent.Client
}

// GetHighestBlock implements Repository.
func (r *RepositoryEnt) GetHighestBlock(ctx context.Context) (*model.Block, error) {
	entBlock, err := r.client.Block.Query().
		Order(ent.Desc("height")).
		First(ctx)
	if err != nil {
		return nil, r.logger.Errorf("failed to get highest block: %v", err)
	}
	return &model.Block{
		Hash:     entBlock.Hash,
		Height:   entBlock.ID,
		Time:     entBlock.Time,
		TotalTxs: entBlock.TotalTxs,
		NumTxs:   entBlock.NumTxs,
	}, nil
}

// AddBlock implements Repository.
func (r *RepositoryEnt) AddBlock(ctx context.Context, block *model.Block) (bool, error) {
	// Convert model.Block to ent.Block
	// Use the ent client to create the block in the database
	_, err := r.client.Block.Create().
		SetHash(block.Hash).
		SetID(block.Height). // Use Height as ID for uniqueness")
		// SetHeight(block.Height).
		SetTime(block.Time).
		SetTotalTxs(block.TotalTxs).
		SetNumTxs(block.NumTxs).
		SetCreatedAt(time.Now()).
		Save(ctx)

	if ent.IsConstraintError(err) {
		// If the block already exists, we can ignore the error
		return true, nil
	} else if err != nil {
		return false, r.logger.Errorf("failed to add block: %v", err)
	}

	return false, nil
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
			SetID(block.Height).
			SetTime(block.Time).
			SetTotalTxs(block.TotalTxs).
			SetNumTxs(block.NumTxs).
			SetCreatedAt(time.Now())
	}

	_, err := r.client.Block.CreateBulk(bulk...).Save(ctx)
	if err != nil {
		return r.logger.Errorf("failed to add blocks: %v", err)
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
		SetBlockID(blockNum).
		SetCreatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return r.logger.Errorf("failed to add transaction: %v", err)
	}

	return nil
}

// AddTransactions implements Repository.
func (r *RepositoryEnt) AddTransactions(ctx context.Context, blockNum int, txs []model.Transaction) error {
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
			SetBlockID(blockNum).
			SetCreatedAt(time.Now())
	}

	_, err := r.client.Transaction.CreateBulk(bulk...).Save(ctx)
	if ent.IsConstraintError(err) {
		// If the transactions already exist, we can ignore the error
	} else if err != nil {
		return r.logger.Errorf("failed to add transactions: %v", err)
	}

	return nil
}

// GetBlock implements Repository.
func (r *RepositoryEnt) GetBlock(ctx context.Context, blockNum int) (*model.Block, error) {
	entBlock, err := r.client.Block.Query().
		Where(block.IDEQ(blockNum)).
		Only(ctx)
	if err != nil {
		return nil, r.logger.Errorf("failed to get block %d: %v", blockNum, err)
	}

	return &model.Block{
		Hash:     entBlock.Hash,
		Height:   entBlock.ID,
		Time:     entBlock.Time,
		TotalTxs: entBlock.TotalTxs,
		NumTxs:   entBlock.NumTxs,
	}, nil
}

// GetBlocks implements Repository.
func (r *RepositoryEnt) GetBlocks(ctx context.Context, offset int, limit int) ([]model.Block, error) {
	entBlocks, err := r.client.Block.Query().
		Offset(offset).
		Limit(limit).
		Order(ent.Desc("height")).
		All(ctx)
	if err != nil {
		return nil, r.logger.Errorf("failed to get blocks (offset=%d, limit=%d): %v", offset, limit, err)
	}

	blocks := make([]model.Block, len(entBlocks))
	for i, entBlock := range entBlocks {
		blocks[i] = model.Block{
			Hash:     entBlock.Hash,
			Height:   entBlock.ID,
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
		return nil, r.logger.Errorf("failed to get transaction %s: %v", txHash, err)
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
func (r *RepositoryEnt) GetTransactions(ctx context.Context, blockNum int, offset int, limit int) ([]model.Transaction, error) {
	entTxs, err := r.client.Transaction.Query().
		Where(transaction.BlockHeightEQ(blockNum)).
		Offset(offset).
		Limit(limit).
		Order(ent.Asc("index")).
		All(ctx)
	if err != nil {
		return nil, r.logger.Errorf("failed to get transactions for block %d (offset=%d, limit=%d): %v", blockNum, offset, limit, err)
	}

	txs := make([]model.Transaction, len(entTxs))
	for i, entTx := range entTxs {
		txs[i] = model.Transaction{
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

// AddAccount implements Repository.
func (r *RepositoryEnt) AddAccount(ctx context.Context, account *model.Account) error {
	_, err := r.client.Account.Create().
		SetID(account.Address).
		SetToken(account.Token).
		SetAmount(account.Amount).
		Save(ctx)
	if err != nil {
		return r.logger.Errorf("failed to add account %s: %v", account.Address, err)
	}

	return nil
}

// GetAccount implements Repository.
func (r *RepositoryEnt) GetAccount(ctx context.Context, address string, token string) (*model.Account, error) {
	accountQuery := r.client.Account.Query()
	if address != "" {
		accountQuery = accountQuery.Where(
			account.IDEQ(address),
		)
	}
	if token != "" {
		accountQuery = accountQuery.Where(
			account.TokenEQ(token),
		)
	}
	entAccount, err := accountQuery.Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil // Account not found
		}
		return nil, r.logger.Errorf("failed to get account %s: %v", address, err)
	}

	return &model.Account{
		Address: address,
		Token:   token,
		Amount:  entAccount.Amount,
	}, nil
}

// GetAccounts implements Repository.
func (r *RepositoryEnt) GetAccounts(ctx context.Context, address string, token string) ([]model.Account, error) {
	accountQuery := r.client.Account.Query()
	if address != "" {
		accountQuery = accountQuery.Where(account.IDEQ(address))
	}
	if token != "" {
		accountQuery = accountQuery.Where(account.TokenEQ(token))
	}
	accounts, err := accountQuery.All(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil // No accounts found
		}
		return nil, r.logger.Errorf("failed to get accounts for %s: %v", address, err)
	}

	accountModels := make([]model.Account, len(accounts))
	for i, entAccount := range accounts {
		accountModels[i] = model.Account{
			Address: entAccount.ID,
			Token:   entAccount.Token,
			Amount:  entAccount.Amount,
		}
	}

	return accountModels, nil
}

// GetBalances implements Repository.
func (r *RepositoryEnt) GetBalances(ctx context.Context, address string) ([]model.TokenBalance, error) {
	// group by token
	var balances []model.TokenBalance
	var accountQuery *ent.AccountGroupBy
	if address == "" {
		accountQuery = r.client.Account.Query().
			GroupBy(account.FieldToken)
	} else {
		accountQuery = r.client.Account.Query().
			Where(account.IDEQ(address)).
			GroupBy(account.FieldToken)
	}
	err := accountQuery.Aggregate(ent.As(ent.Sum(account.FieldAmount), "amount")).
		Scan(ctx, &balances)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil // No balances found
		}
		return nil, r.logger.Errorf("failed to get balances for %s: %v", address, err)
	}

	return balances, nil
}

// IncrementAccountBalance implements Repository.
func (r *RepositoryEnt) IncrementAccountBalance(ctx context.Context, address string, token string, amount int64) error {
	// Use a transaction to ensure atomicity
	_, err := r.client.Account.Update().
		Where(
			account.IDEQ(address),
			account.TokenEQ(token),
		).
		AddAmount(float64(amount)).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return r.logger.Errorf("account %s with token %s not found: %v", address, token, err)
		}
		return r.logger.Errorf("failed to increment account balance for %s: %v", address, err)
	}

	return nil
}

// AddTransfer implements Repository.
func (r *RepositoryEnt) AddTransfer(ctx context.Context, tx *model.Transaction, transfer *model.Transfer) error {
	createTransfer := r.client.Transfer.Create().
		SetHash(tx.Hash).
		SetFunc(strings.ToLower(transfer.Func)).
		SetToken(transfer.Token).
		SetAmount(transfer.Amount).
		SetDenom(transfer.Denom).
		SetCreatedAt(time.Now())

	if transfer.FromAddress != "" {
		createTransfer = createTransfer.SetFromAddress(transfer.FromAddress)
	}
	if transfer.ToAddress != "" {
		createTransfer = createTransfer.SetToAddress(transfer.ToAddress)
	}
	_, err := createTransfer.Save(ctx)
	if err != nil {
		return r.logger.Errorf("failed to add transfer from %s to %s: %v", transfer.FromAddress, transfer.ToAddress, err)
	}

	return nil
}

// AddTransfers implements Repository.
func (r *RepositoryEnt) AddTransfers(ctx context.Context, tx *model.Transaction, transfers []model.Transfer) error {
	_, err := r.client.Transfer.CreateBulk(
		func() []*ent.TransferCreate {
			bulk := make([]*ent.TransferCreate, len(transfers))
			for i, transfer := range transfers {
				bulk[i] = r.client.Transfer.Create().
					SetHash(tx.Hash).
					SetFunc(strings.ToLower(transfer.Func)).
					SetToken(transfer.Token).
					SetAmount(transfer.Amount).
					SetDenom(transfer.Denom).
					SetCreatedAt(time.Now())

				if transfer.FromAddress != "" {
					bulk[i].SetFromAddress(transfer.FromAddress)
				}
				if transfer.ToAddress != "" {
					bulk[i].SetToAddress(transfer.ToAddress)
				}
			}
			return bulk
		}()...).Save(ctx)
	if err != nil {
		return r.logger.Errorf("failed to add transfers: %v", err)
	}
	return nil
}

// GetTransfers implements Repository.
func (r *RepositoryEnt) GetTransfers(ctx context.Context, fromAccountOr string, toAccountOr string, token string) ([]model.Transfer, error) {
	transferQuery := r.client.Transfer.Query()
	if fromAccountOr != "" {
		transferQuery = transferQuery.Where(
			transfer.Or(transfer.FromAddressEQ(fromAccountOr)),
		)
	}
	if toAccountOr != "" {
		transferQuery = transferQuery.Where(
			transfer.Or(transfer.ToAddressEQ(toAccountOr)),
		)
	}
	if token != "" {
		transferQuery = transferQuery.Where(
			transfer.TokenEQ(token),
		)
	}
	transfers, err := transferQuery.Where(transfer.Func("transfer")).All(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil // Transfer not found
		}
		return nil, r.logger.Errorf("failed to get transfer from %s to %s: %v", fromAccountOr, toAccountOr, err)
	}

	transferModels := make([]model.Transfer, len(transfers))
	for i, entTransfer := range transfers {
		transferModels[i] = model.Transfer{
			FromAddress: entTransfer.FromAddress,
			ToAddress:   entTransfer.ToAddress,
			Token:       entTransfer.Token,
			Amount:      entTransfer.Amount,
			Denom:       entTransfer.Denom,
			CreatedAt:   entTransfer.CreatedAt,
		}
	}
	return transferModels, nil
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
		Log:   modelResponse.Log,
		Info:  modelResponse.Info,
		Error: modelResponse.Error,
		Data:  modelResponse.Data,
		Events: func() []schema.Event {
			events := make([]schema.Event, len(modelResponse.Events))
			for i, event := range modelResponse.Events {
				events[i] = schema.Event{
					Type:    event.Type,
					Func:    event.Func,
					PkgPath: event.PkgPath,
					Attrs:   event.Attrs,
				}
			}
			return events
		}(),
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
		Log:   schemaResponse.Log,
		Info:  schemaResponse.Info,
		Error: schemaResponse.Error,
		Data:  schemaResponse.Data,
		Events: func(response schema.Response) []model.Event {
			events := make([]model.Event, len(response.Events))
			for i, event := range response.Events {
				events[i] = model.Event{
					Type:    event.Type,
					Func:    event.Func,
					PkgPath: event.PkgPath,
					Attrs:   event.Attrs,
				}
			}
			return events
		}(schemaResponse),
	}
}
