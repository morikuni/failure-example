package controller

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/morikuni/failure-example/simple-crud/service"

	"github.com/morikuni/failure"
	"github.com/morikuni/failure-example/simple-crud/errors"
	"github.com/morikuni/failure-example/simple-crud/model"
)

type Controller struct {
	service service.Service
	logger  *log.Logger
	port    string
}

func New(service service.Service, logger *log.Logger, port string) *Controller {
	return &Controller{service, logger, port}
}

func (c *Controller) Run(ctx context.Context, ready chan<- struct{}) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/create", c.handleError(c.create))
	mux.HandleFunc("/read", c.handleError(c.read))
	mux.HandleFunc("/update", c.handleError(c.update))
	mux.HandleFunc("/delete", c.handleError(c.delete))

	server := http.Server{
		Handler: mux,
	}

	l, err := net.Listen("tcp", ":"+c.port)
	if err != nil {
		return failure.Wrap(err)
	}

	go func() {
		err := server.Serve(l)
		if err != nil && err != http.ErrServerClosed {
			c.logger.Printf("[ERROR] %v\n", err)
		}
	}()

	if ready != nil {
		close(ready)
	}
	<-ctx.Done()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return failure.Wrap(server.Shutdown(ctx))
}

func (c *Controller) create(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	key, err := model.NewKey(r.FormValue("key"))
	if err != nil {
		return failure.Wrap(err)
	}
	value, err := model.NewValue(r.FormValue("value"))
	if err != nil {
		return failure.Wrap(err)
	}

	err = c.service.Create(ctx, key, value)
	if err != nil {
		return failure.Wrap(err)
	}

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "created")
	return nil
}

func (c *Controller) read(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	key, err := model.NewKey(r.FormValue("key"))
	if err != nil {
		return failure.Wrap(err)
	}

	value, err := c.service.Read(ctx, key)
	if err != nil {
		return failure.Wrap(err)
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%d", value)
	return nil
}

func (c *Controller) update(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	key, err := model.NewKey(r.FormValue("key"))
	if err != nil {
		return failure.Wrap(err)
	}
	value, err := model.NewValue(r.FormValue("value"))
	if err != nil {
		return failure.Wrap(err)
	}

	err = c.service.Update(ctx, key, value)
	if err != nil {
		return failure.Wrap(err)
	}

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "updated")
	return nil
}

func (c *Controller) delete(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	key, err := model.NewKey(r.FormValue("key"))
	if err != nil {
		return failure.Wrap(err)
	}

	err = c.service.Delete(ctx, key)
	if err != nil {
		return failure.Wrap(err)
	}

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "deleted")
	return nil
}

func (c *Controller) handleError(f func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := f(w, r)
		if err == nil {
			return
		}

		var shouldPrintDetail bool
		code, ok := failure.CodeOf(err)
		if !ok {
			shouldPrintDetail = true
		}

		status := httpStatus(code)
		if status%100 == 5 {
			shouldPrintDetail = true
		}
		w.WriteHeader(status)

		msg, ok := failure.MessageOf(err)
		if ok {
			io.WriteString(w, msg)
		} else {
			io.WriteString(w, http.StatusText(status))
			shouldPrintDetail = true
		}

		if shouldPrintDetail {
			c.logger.Printf("[ERROR] %+v\n", err)
		} else {
			c.logger.Printf("[ERROR] %v\n", err)
		}
	}
}

func httpStatus(code failure.Code) int {
	switch code {
	case errors.InvalidArgument:
		return http.StatusBadRequest
	case errors.NotFound:
		return http.StatusNotFound
	case errors.AlreadyExist:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
