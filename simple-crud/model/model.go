package model

import (
	"strconv"

	"github.com/morikuni/failure"
	"github.com/morikuni/failure-example/simple-crud/errors"
)

type Key string

func NewKey(key string) (Key, error) {
	const maxLen = 256
	if len(key) > maxLen {
		return "", failure.New(errors.InvalidArgument,
			failure.Messagef("Key must be less than %d bytes.", maxLen),
		)
	}

	return Key(key), nil
}

type Value int64

func NewValue(value string) (Value, error) {
	i, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, failure.New(errors.InvalidArgument,
			failure.Message("Value must be number."),
		)
	}

	return Value(i), nil
}
