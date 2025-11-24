package domain

import "errors"

var (
	// ErrUserNotFound is returned when a user cannot be found.
	ErrUserNotFound = errors.New("user not found")

	// ErrItemNotFound is returned when an item cannot be found.
	ErrItemNotFound = errors.New("item not found")

	// ErrInsufficientQuantity is returned when a user does not have enough of an item.
	ErrInsufficientQuantity = errors.New("insufficient quantity")

	// ErrInvalidInput is returned when input validation fails.
	ErrInvalidInput = errors.New("invalid input")

	// ErrInvalidPlatform is returned when a platform is not supported.
	ErrInvalidPlatform = errors.New("invalid platform")
)
