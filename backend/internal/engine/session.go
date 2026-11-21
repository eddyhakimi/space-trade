package engine

import (
	"context"
	"crypto/rand"
	"errors"

	"github.com/eddyhakimi/space-trade/backend/internal/game"
)

var ErrUnknownToken = errors.New("unknown session, please join with a name")

// Join returns the player for a connection and their session token:
//
//   - a known token resumes its player;
//   - otherwise a name that belongs to a player with no open connection takes
//     that player over with a new token (after Leave, a lost welcome, or a
//     second tab holding an outdated token). Names are the identity among
//     friends; a player who is connected right now can't be taken over;
//   - otherwise a new player is created with that name.
func (e *Engine) Join(ctx context.Context, name, token string) (game.PlayerID, string, error) {
	var id game.PlayerID
	err := e.exec(ctx, func() (bool, error) {
		if pid, ok := e.tokens[token]; ok && token != "" {
			id = pid
			return false, nil
		}
		if name == "" {
			return false, ErrUnknownToken
		}
		if p, ok := e.world.PlayerByName(name); ok {
			if e.conns[p.ID] > 0 {
				return false, game.ErrNameTaken
			}
			id, token = p.ID, e.newToken(p.ID)
			return false, nil
		}
		p, err := e.world.AddPlayer(name)
		if err != nil {
			return false, err
		}
		id, token = p.ID, e.newToken(p.ID)
		return true, nil
	})
	if err != nil {
		return "", "", err
	}
	return id, token, nil
}

// newToken issues a session token for pid. Older tokens stay valid, so other
// tabs of the same player keep working.
func (e *Engine) newToken(pid game.PlayerID) string {
	token := rand.Text()
	e.tokens[token] = pid
	return token
}

// Subscribe registers a connection of player pid, marks it online and
// starts streaming snapshots (the first one arrives right away).
func (e *Engine) Subscribe(ctx context.Context, pid game.PlayerID) (*Subscription, error) {
	var sub *Subscription
	err := e.exec(ctx, func() (bool, error) {
		if err := e.world.SetOnline(pid, true); err != nil {
			return false, err
		}
		c := make(chan game.Snapshot, subBuffer)
		sub = &Subscription{C: c, c: c, player: pid}
		e.subs[sub] = struct{}{}
		e.conns[pid]++
		return true, nil
	})
	return sub, err
}

// Unsubscribe ends a connection. The player goes offline when it was their last one.
func (e *Engine) Unsubscribe(sub *Subscription) {
	_ = e.exec(context.Background(), func() (bool, error) {
		e.drop(sub)
		e.conns[sub.player]--
		if e.conns[sub.player] > 0 {
			return false, nil
		}
		delete(e.conns, sub.player)
		return true, e.world.SetOnline(sub.player, false)
	})
}
