package game

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type PlayerID string

const MaxNameLen = 20

type Player struct {
	ID     PlayerID `json:"id"`
	Name   string   `json:"name"`
	Online bool     `json:"online"`
	// Credits are whole units, owned by the player (not the ship).
	Credits int `json:"credits"`
	// Heat is the remaining wanted time in ticks (see wanted.go).
	Heat int `json:"heat"`
}

// AddPlayer creates a player with one ship docked at a random station.
func (w *World) AddPlayer(name string) (*Player, error) {
	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n == 0 || n > MaxNameLen {
		return nil, ErrInvalidName
	}
	if _, taken := w.PlayerByName(name); taken {
		return nil, ErrNameTaken
	}

	w.nextPlayer++
	p := &Player{ID: PlayerID(fmt.Sprintf("p%d", w.nextPlayer)), Name: name, Credits: StartingCredits}
	w.Players[p.ID] = p

	start := w.Stations[w.rng.IntN(len(w.Stations))]
	w.newShip(p, start)
	return p, nil
}

// PlayerByName finds a player by name (trimmed, case-insensitive), the same
// way AddPlayer decides that a name is taken.
func (w *World) PlayerByName(name string) (*Player, bool) {
	name = strings.TrimSpace(name)
	for _, p := range w.Players {
		if strings.EqualFold(p.Name, name) {
			return p, true
		}
	}
	return nil, false
}

// SetOnline marks a player as connected or not.
func (w *World) SetOnline(playerID PlayerID, online bool) error {
	p, ok := w.Players[playerID]
	if !ok {
		return ErrUnknownPlayer
	}
	p.Online = online
	return nil
}
