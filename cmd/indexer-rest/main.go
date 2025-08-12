package main

import (
	"context"
	"gno.land-block-indexer/cmd/indexer-rest/controller"
	"gno.land-block-indexer/lib/log"
)

func main() {
	ctx := context.Background()
	logger := log.NewLogger()
	controller := controller.NewController(logger)
	err := controller.Run(ctx)
	if err != nil {
		logger.Fatalf("Failed to run controller: %v", err)
	}

	// Wait Signal C-c
	<-ctx.Done()
	if err := ctx.Err(); err != nil {
		// Handle the error, e.g., log it
		// In a real application, you might want to handle this more gracefully
		panic("application terminated with error: " + err.Error())
	} else {
		// Application terminated gracefully
	}
}
