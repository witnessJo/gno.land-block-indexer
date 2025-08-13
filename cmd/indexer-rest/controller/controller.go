package controller

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/coocood/freecache"
	"github.com/gin-gonic/gin"
	"gno.land-block-indexer/cmd/indexer-rest/service"
	"gno.land-block-indexer/lib/log"
	"gno.land-block-indexer/repository"
)

type Controller struct {
	logger     log.Logger
	localCache *freecache.Cache
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

	cacheSize := 100 * 1024 * 1024 // 100 MB
	localCache := freecache.NewCache(cacheSize)
	if localCache == nil {
		logger.Fatalf("Failed to create local cache with size %d bytes", cacheSize)
	}

	return &Controller{
		engine:     engine,
		logger:     logger,
		service:    service,
		listenHost: "localhost",
		listenPort: 8080,
		localCache: localCache,
	}
}

func (c *Controller) getMemcacheKey(ctx *gin.Context) string {
	return fmt.Sprintf("%s?%s", ctx.Request.URL.Path, ctx.Request.URL.RawQuery)
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (c *Controller) Run(ctx context.Context) error {
	c.engine.Use(func(gCtx *gin.Context) {
		if gCtx.Request.Method != "GET" {
			c.logger.Infof("Skipping caching for non-GET request: %s %s", gCtx.Request.Method, gCtx.Request.URL.Path)
			gCtx.Next()
			return
		}
		key := c.getMemcacheKey(gCtx)
		if value, found := c.findFromLocalCache(key); found {
			c.logger.Infof("Cache hit for key: %s", key)
			gCtx.Data(200, "application/json", value)
			gCtx.Abort()
			return
		}
		gCtx.Set("cached", false)
		gCtx.Set("memcacheKey", key)
	})

	c.engine.Use(func(gCtx *gin.Context) {
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: gCtx.Writer}
		gCtx.Writer = blw

		gCtx.Next()

		cached, ok := gCtx.Get("cached")
		if ok && cached.(bool) {
			return
		}
		key, ok := gCtx.Get("memcacheKey")
		if !ok {
			c.logger.Errorf("No memcache key found in context")
			return
		}

		if gCtx.Writer.Status() == 200 && blw.body.Len() > 0 {
			data := blw.body.Bytes()
			if err := c.localCache.Set([]byte(key.(string)), data, 60); err != nil {
				c.logger.Errorf("Failed to set cache for key %s: %v", key, err)
			} else {
				c.logger.Infof("Stored response in local cache for key: %s", key)
			}
		}
	})

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

func (c *Controller) findFromLocalCache(key string) ([]byte, bool) {
	value, err := c.localCache.Get([]byte(key))
	if err != nil {
		if err == freecache.ErrNotFound {
			return nil, false
		}
		c.logger.Errorf("Error retrieving from local cache: %v", err)
		return nil, false
	}
	return value, true
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
