package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/morikuni/failure-example/simple-crud/controller"
	"github.com/morikuni/failure-example/simple-crud/database"
	"github.com/morikuni/failure-example/simple-crud/service"
	"golang.org/x/sync/errgroup"
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

	ctx, wm := WithContext(ctx)
	dbSig := wm.GoReady(ctx, db.Run)
	controllerSig := wm.GoReady(ctx, c.Run, dbSig)
	wm.Go(ctx, func(ctx context.Context) error {
		logger.Println("[INFO] controller is ready")
		return nil
	}, controllerSig)

	if err := wm.Wait(); err != nil {
		logger.Printf("[ERROR] %v\n", err)
	}
}

type ReadySignal = <-chan struct{}

type WorkerManager struct {
	eg errgroup.Group
}

func WithContext(ctx context.Context) (context.Context, *WorkerManager) {
	eg, ctx := errgroup.WithContext(ctx)
	return ctx, &WorkerManager{*eg}
}

func (wm *WorkerManager) GoReady(ctx context.Context, worker func(ctx context.Context, ready chan<- struct{}) error, wait ...ReadySignal) ReadySignal {
	c := make(chan struct{})
	wm.eg.Go(func() error {
		for _, w := range wait {
			select {
			case <-w:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return worker(ctx, c)
	})
	return c
}

func (wm *WorkerManager) Go(ctx context.Context, worker func(ctx context.Context) error, wait ...ReadySignal) {
	wm.eg.Go(func() error {
		for _, w := range wait {
			select {
			case <-w:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return worker(ctx)
	})
}

func (wm *WorkerManager) Wait() error {
	return wm.eg.Wait()
}
