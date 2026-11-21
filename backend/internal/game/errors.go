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
	ErrRouteFull     = errors.New("route is full (max 10 stops)")

	ErrBadQuantity    = errors.New("quantity must be at least 1")
	ErrUnknownGood    = errors.New("unknown good")
	ErrNotDocked      = errors.New("dock at a station to trade")
	ErrNotTradedHere  = errors.New("this station doesn't trade that good")
	ErrNotForSale     = errors.New("this station doesn't sell that good")
	ErrCargoFull      = errors.New("not enough cargo space")
	ErrNoCredits      = errors.New("not enough credits")
	ErrNotEnoughCargo = errors.New("you don't have that much in your hold")
)
