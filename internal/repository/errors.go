package repository

import (
	"errors"

	"gorm.io/gorm"
)

// IsNotFound reports whether the error means the record was not found.
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
