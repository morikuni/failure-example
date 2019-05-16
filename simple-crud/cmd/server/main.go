package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/morikuni/failure"

	"github.com/morikuni/failure-example/simple-crud/service"
	"golang.org/x/sync/errgroup"

	"github.com/morikuni/failure-example/simple-crud/controller"
	"github.com/morikuni/failure-example/simple-crud/database"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)
	db := database.NewMySQL("user:pass@tcp(127.0.0.1:3306)/main")
	s := service.New(db)
	c := controller.New(s, logger, "8080")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sig := make(chan os.Signal)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		cancel()
	}()

	var (
		dbReady         = make(chan struct{})
		controllerReady = make(chan struct{})
	)

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return failure.Wrap(db.Run(ctx, dbReady))
	})
	eg.Go(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-dbReady:
		}

		return failure.Wrap(c.Run(ctx, controllerReady))
	})
	eg.Go(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-controllerReady:
			logger.Println("[INFO] controller is ready")
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		logger.Printf("[ERROR] %v\n", err)
	}
}
