package repository

import "errors"

var (
	ErrMaterialUnitMismatch = errors.New("material default unit does not match inbound unit")
	ErrInsufficientStock    = errors.New("insufficient stock")
)
