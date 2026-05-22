package review

import "errors"

var (
	ErrInvalidInput      = errors.New("invalid review input")
	ErrNotFound          = errors.New("review not found")
	ErrDuplicateReview   = errors.New("user already reviewed this product")
	ErrNotEligible       = errors.New("user is not eligible to review this product")
	ErrProductNotFound   = errors.New("product rating not found")
)
