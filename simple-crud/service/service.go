package service

import (
	"context"
	"io/ioutil"
	"log"

	"github.com/go-sql-driver/mysql"

	"github.com/morikuni/failure"
	"github.com/morikuni/failure-example/simple-crud/database"
	"github.com/morikuni/failure-example/simple-crud/errors"
	"github.com/morikuni/failure-example/simple-crud/model"
)

type Service interface {
	Create(ctx context.Context, key model.Key, value model.Value) error
	Read(ctx context.Context, key model.Key) (model.Value, error)
	Update(ctx context.Context, key model.Key, value model.Value) error
	Delete(ctx context.Context, key model.Key) error
}

func New(db database.Database) Service {
	return &service{db}
}

type service struct {
	db database.Database
}

func (s *service) Create(ctx context.Context, key model.Key, value model.Value) error {
	context := func() failure.Context { return failure.Context{"key": string(key)} }

	_, err := s.db.Get(ctx, key)
	if err == nil {
		return failure.New(errors.AlreadyExist,
			failure.Message("Specified key already exists. Use update for existing key."),
			context(),
		)
	}
	if !failure.Is(err, errors.NotFound) {
		return failure.Wrap(err,
			context(),
		)
	}

	err = s.db.Put(ctx, key, value)
	if err != nil {
		return failure.Wrap(err,
			context(),
		)
	}

	return nil
}

func (s *service) Read(ctx context.Context, key model.Key) (model.Value, error) {
	context := func() failure.Context { return failure.Context{"key": string(key)} }

	v, err := s.db.Get(ctx, key)
	if err != nil {
		ws := []failure.Wrapper{context()}
		if failure.Is(err, errors.NotFound) {
			ws = append(ws, failure.Message("Specified key does not exist."))
		}
		return 0, failure.Wrap(err, ws...)
	}

	return v, nil
}

func (s *service) Update(ctx context.Context, key model.Key, value model.Value) error {
	context := func() failure.Context { return failure.Context{"key": string(key)} }

	_, err := s.db.Get(ctx, key)
	if err != nil {
		ws := []failure.Wrapper{context()}
		if failure.Is(err, errors.NotFound) {
			ws = append(ws, failure.Message("Specified key does not exist."))
		}
		return failure.Wrap(err, ws...)
	}

	err = s.db.Put(ctx, key, value)
	if err != nil {
		return failure.Wrap(err,
			context(),
		)
	}

	return nil
}

func (s *service) Delete(ctx context.Context, key model.Key) error {
	context := func() failure.Context { return failure.Context{"key": string(key)} }

	if err := s.db.Delete(ctx, key); err != nil {
		ws := []failure.Wrapper{context()}
		if failure.Is(err, errors.NotFound) {
			ws = append(ws, failure.Message("Specified key does not exist."))
		}
		return failure.Wrap(err, ws...)
	}

	return nil
}

func init() {
	mysql.SetLogger(log.New(ioutil.Discard, "", 0))
}
