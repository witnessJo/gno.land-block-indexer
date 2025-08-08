package main

import (
	"context"

	"gno.land-block-indexer/cmd/block-synchronizer/controller"
)

func main() {
	ctx := context.Background()
	controller := controller.NewController()
	controller.Run(ctx)

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
