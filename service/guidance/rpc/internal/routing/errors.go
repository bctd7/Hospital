package routing

import "errors"

var (
	ErrInvalid     = errors.New("invalid route request")
	ErrUnavailable = errors.New("route provider unavailable")
	ErrNoRoute     = errors.New("walking route not found")
	ErrProvider    = errors.New("route provider failed")
)
