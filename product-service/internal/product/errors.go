package product

import "errors"

var (
	ErrInvalidInput        = errors.New("invalid product input")
	ErrNotFound            = errors.New("product not found")
	ErrInsufficientStock   = errors.New("insufficient stock")
	ErrReservationNotFound = errors.New("reservation not found")
)
