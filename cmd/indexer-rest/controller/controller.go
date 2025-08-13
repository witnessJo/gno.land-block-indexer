package controller

import (
	"context"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gno.land-block-indexer/cmd/indexer-rest/service"
	"gno.land-block-indexer/lib/log"
	"gno.land-block-indexer/repository"
)

type Controller struct {
	logger     log.Logger
	listenHost string
	listenPort int
	service    service.Service
	engine     *gin.Engine
}

func NewController(logger log.Logger) *Controller {
	engine := gin.Default()
	repo := repository.NewRepositoryEnt(logger, &repository.RepositoryEntConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		Database: "postgres",
	})
	service := service.NewService(logger, repo)

	return &Controller{
		engine:     engine,
		logger:     logger,
		service:    service,
		listenHost: "localhost",
		listenPort: 8080,
	}
}

func (c *Controller) Run(ctx context.Context) error {
	c.engine.GET("/tokens/*any", c.handleTokenRoutes)
	// Start the HTTP server
	address := c.listenHost + ":" + strconv.Itoa(c.listenPort)
	if err := c.engine.Run(address); err != nil {
		return c.logger.Errorf("Failed to start HTTP server: %v", err)
	}

	c.logger.Infof("HTTP server started on %s", address)
	c.logger.Infof("Listening on %s:%d", c.listenHost, c.listenPort)
	c.logger.Infof("Repository connected to %s:%d", c.listenHost, c.listenPort)

	return nil
}

func (c *Controller) handleTokenRoutes(gCtx *gin.Context) {
	path := gCtx.Param("any")

	switch {
	case path == "/balances":
		c.GetTokenBalances(gCtx)
	case path == "/transfer-history":
		c.GetTransferHistory(gCtx)
	case strings.HasSuffix(path, "/balances"):
		tokenPath := strings.TrimSuffix(path, "/balances")
		if tokenPath == "" {
			c.logger.Errorf("Token path is empty")
			gCtx.JSON(400, gin.H{"error": "Token path cannot be empty"})
			return
		}
		if strings.HasPrefix(tokenPath, "/") {
			tokenPath = strings.TrimPrefix(tokenPath, "/")
		}
		c.GetTokenAccountBalances(gCtx, tokenPath)
	default:
		gCtx.JSON(404, gin.H{"error": "endpoint not found"})
	}
}

func (c *Controller) GetTokenBalances(gCtx *gin.Context) {
	ctx := gCtx.Request.Context()
	var request struct {
		Address string `form:"address"`
	}
	if err := gCtx.ShouldBindQuery(&request); err != nil {
		c.logger.Errorf("Failed to bind request: %v", err)
		gCtx.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	accounts, err := c.service.GetTokenBalances(ctx, request.Address)
	if err != nil {
		c.logger.Errorf("Failed to get account balances: %v", err)
		gCtx.JSON(500, gin.H{"error": "Failed to get account balances"})
		return
	}

	type Balance struct {
		TokenPath string `json:"tokenPath"`
		Amount    int64  `json:"amount"`
	}
	var response struct {
		Balances []Balance `json:"balances"`
	}
	for _, account := range accounts {
		response.Balances = append(response.Balances, Balance{
			TokenPath: account.Token,
			Amount:    int64(account.Amount),
		})
	}
	if len(response.Balances) == 0 {
		response.Balances = []Balance{}
	}

	gCtx.JSON(200, response)
}

func (c *Controller) GetTokenAccountBalances(gCtx *gin.Context, tokenPath string) {
	ctx := gCtx.Request.Context()
	address := gCtx.Query("address")
	tokenAccountBalances, err := c.service.GetTokenAccountBalances(ctx, tokenPath, address)
	if err != nil {
		return
	}

	type TokenAccountBalance struct {
		Address   string `json:"address"`
		TokenPath string `json:"tokenPath"`
		Amount    int64  `json:"amount"`
	}
	var response struct {
		AccountBalances []TokenAccountBalance `json:"accountBalances"`
	}
	for _, account := range tokenAccountBalances {
		response.AccountBalances = append(response.AccountBalances, TokenAccountBalance{
			Address:   account.Address,
			TokenPath: account.Token,
			Amount:    int64(account.Amount),
		})
	}
	if len(response.AccountBalances) == 0 {
		response.AccountBalances = []TokenAccountBalance{}
	}

	gCtx.JSON(200, response)
}

func (c *Controller) GetTransferHistory(gCtx *gin.Context) {
	ctx := gCtx.Request.Context()
	var request struct {
		Address string `form:"address"`
	}
	transferHistories, err := c.service.GetTransferHistory(ctx, request.Address)
	if err != nil {
		return
	}

	type Transfer struct {
		FromAddress string `json:"fromAddress"`
		ToAddress   string `json:"toAddress"`
		Token       string `json:"token"`
		Amount      int64  `json:"amount"`
	}
	var response struct {
		Transfer []Transfer `json:"transfers"`
	}

	for _, transferHistory := range transferHistories {
		response.Transfer = append(response.Transfer, Transfer{
			FromAddress: transferHistory.FromAddress,
			ToAddress:   transferHistory.ToAddress,
			Token:       transferHistory.Token,
			Amount:      int64(transferHistory.Amount),
		})
	}
	if len(response.Transfer) == 0 {
		response.Transfer = []Transfer{}
	}

	gCtx.JSON(200, response)
}
