package repository

import "errors"

var (
	ErrCategoryInUse     = errors.New("category is in use")
	ErrCategoryNameExists = errors.New("category name already exists")
	ErrInvalidCategoryID = errors.New("invalid category id")
)
