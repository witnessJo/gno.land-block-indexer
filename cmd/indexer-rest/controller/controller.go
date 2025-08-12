package controller

import (
	"context"
	"strconv"

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
	c.engine.GET("/account/balances", c.GetBalances)
	c.engine.GET("/token/:token_path/balances", c.GetTokenBalances)
	c.engine.GET("/transfer/history", c.GetTransferHistory)

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

func (c *Controller) GetBalances(gCtx *gin.Context) {
	ctx := gCtx.Request.Context()
	var request struct {
		Address string `json:"address" binding:"required"`
	}
	accounts, err := c.service.GetTokenBalances(ctx, request.Address)
	if err != nil {
		c.logger.Errorf("Failed to get account balances: %v", err)
		gCtx.JSON(500, gin.H{"error": "Failed to get account balances"})
		return
	}

	var response struct {
		Balances []struct {
			TokenPath string
			Amount    int64
		}
	}
	for _, account := range accounts {
		response.Balances = append(response.Balances, struct {
			TokenPath string
			Amount    int64
		}{
			TokenPath: account.Token,
			Amount:    int64(account.Amount),
		})
	}

	gCtx.JSON(200, response)
	return
}

func (c *Controller) GetTokenBalances(gCtx *gin.Context) {
	ctx := gCtx.Request.Context()
	// get token_path from URL parameters
	tokenPath := gCtx.Param("token_path")
	if tokenPath == "" {
		c.logger.Errorf("Token path is required")
		gCtx.JSON(400, gin.H{"error": "Token path is required"})
		return
	}
	var request struct {
		TokenPath string `json:"token_path" binding:"required"`
		Address   string `json:"address" binding:"required"`
	}

	token, err := c.service.GetTokenBalances(ctx, request.Address)
	if err != nil {
		return
	}
	if token == nil {
		c.logger.Errorf("Token not found for path: %s", request.TokenPath)
		gCtx.JSON(404, gin.H{"error": "Token not found"})
		return
	}
	gCtx.JSON(200, gin.H{
		"token_path": request.TokenPath,
		"balances":   token,
	})
	return
}

func (c *Controller) GetTransferHistory(gCtx *gin.Context) {
	ctx := gCtx.Request.Context()
	var request struct {
		Address string `json:"address" binding:"required"`
	}
	_, err := c.service.GetTransferHistory(ctx, request.Address)
	if err != nil {
		return
	}
}
