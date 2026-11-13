package vault

import "errors"

var (
	ErrPathNotFound = errors.New("path not found")
	ErrNoData       = errors.New("response contains no data")
)
