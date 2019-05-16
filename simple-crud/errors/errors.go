package errors

import (
	"github.com/morikuni/failure"
)

const (
	InvalidArgument failure.StringCode = "InvalidArgument"
	NotFound        failure.StringCode = "NotFound"
	AlreadyExist    failure.StringCode = "AlreadyExist"
)
