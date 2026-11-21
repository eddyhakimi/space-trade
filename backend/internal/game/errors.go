package game

import "errors"

// Errors returned by World commands. Their messages are shown to players.
var (
	ErrInvalidName   = errors.New("name must be 1-20 characters")
	ErrNameTaken     = errors.New("name is already taken")
	ErrUnknownPlayer = errors.New("unknown player")
	ErrUnknownShip   = errors.New("unknown ship")
	ErrNotYourShip   = errors.New("not your ship")
	ErrOutOfBounds   = errors.New("target is outside the map")
	ErrAlreadyThere  = errors.New("ship is already there")
)
