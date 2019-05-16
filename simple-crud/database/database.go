package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"time"

	"github.com/morikuni/failure-example/simple-crud/errors"

	_ "github.com/go-sql-driver/mysql"
	"github.com/morikuni/failure"
	"github.com/morikuni/failure-example/simple-crud/model"
)

// Database represents a database which has no insert and update method.
// We can use MySQL for inserting and updating but this is just an example.
type Database interface {
	Get(ctx context.Context, key model.Key) (model.Value, error)
	Put(ctx context.Context, key model.Key, value model.Value) error
	Delete(ctx context.Context, key model.Key) error
}

type MySQL struct {
	dsn string

	conn *sql.DB
}

func NewMySQL(dsn string) *MySQL {
	return &MySQL{dsn: dsn}
}

func (db *MySQL) Put(ctx context.Context, key model.Key, value model.Value) error {
	const query = `
INSERT INTO kv (k, v) VALUES (?, ?)
ON DUPLICATE KEY UPDATE v = VALUES(v)
`
	_, err := db.conn.ExecContext(ctx, query, key, value)
	if err != nil {
		return failure.Wrap(err)
	}
	return nil
}

func (db *MySQL) Get(ctx context.Context, key model.Key) (model.Value, error) {
	const query = `
SELECT v FROM kv WHERE k = ?
`
	r := db.conn.QueryRowContext(ctx, query, key)

	var i int64
	if err := r.Scan(&i); err != nil {
		if err == sql.ErrNoRows {
			return 0, failure.Translate(err, errors.NotFound)
		}
		return 0, failure.Wrap(err)
	}

	return model.Value(i), nil
}

func (db *MySQL) Delete(ctx context.Context, key model.Key) error {
	const query = `
DELETE FROM kv WHERE k = ?
`
	r, err := db.conn.ExecContext(ctx, query, key)
	if err != nil {
		return failure.Wrap(err)
	}
	if n, err := r.RowsAffected(); err == nil && n == 0 {
		return failure.New(errors.NotFound)
	}
	return nil
}

func (db *MySQL) Run(ctx context.Context, ready chan<- struct{}) error {
	conn, err := sql.Open("mysql", db.dsn)
	if err != nil {
		return failure.Wrap(err)
	}
	db.conn = conn

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := conn.PingContext(ctx)
		if err == nil {
			break
		}
		if err != driver.ErrBadConn {
			return failure.Wrap(err)
		}
		time.Sleep(time.Second)
	}

	const ddl = `
CREATE TABLE IF NOT EXISTS kv (
	k VARCHAR(256) NOT NULL PRIMARY KEY,
	v BIGINT NOT NULL
)
`
	_, err = conn.ExecContext(ctx, ddl)
	if err != nil {
		return failure.Wrap(err)
	}

	if ready != nil {
		close(ready)
	}

	<-ctx.Done()

	return failure.Wrap(db.conn.Close())
}
