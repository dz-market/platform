package config

import "errors"

var (
	ErrRead       = errors.New("read config")
	ErrParse      = errors.New("parse config")
	ErrDecode     = errors.New("decode config")
	ErrValidate   = errors.New("validate config")
	ErrUnresolved = errors.New("unresolved references")
)
