package repository

import "errors"

// ErrNotFound indicates missing entity in persistence layer
var ErrNotFound = errors.New("repository: entity not found")

func DataNotFoundErr(err error) bool {
	return errors.Is(err, ErrNotFound)
}
