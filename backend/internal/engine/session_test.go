package engine

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/eddyhakimi/space-trade/backend/internal/game"
)

func TestOnlineFollowsConnections(t *testing.T) {
	e := startEngine(t, time.Hour)
	ctx := context.Background()
	pid, sub1 := joinAndSubscribe(t, e, "Alice")
	sub2, err := e.Subscribe(ctx, pid) // second tab
	if err != nil {
		t.Fatal(err)
	}

	online := func() (on bool) {
		e.exec(ctx, func() (bool, error) { // read on the loop goroutine
			on = e.world.Players[pid].Online
			return false, nil
		})
		return on
	}

	e.Unsubscribe(sub1)
	if !online() {
		t.Fatal("Alice offline while a second connection is open")
	}
	if !drainsToClosed(sub1) {
		t.Fatal("unsubscribed channel not closed")
	}
	e.Unsubscribe(sub2)
	if online() {
		t.Fatal("Alice still online after her last connection closed")
	}
}

func TestConcurrentJoins(t *testing.T) {
	e := startEngine(t, time.Millisecond) // ticks interleave with commands
	var wg sync.WaitGroup
	tokens := make([]string, 50)
	errs := make([]error, 50)
	for i := range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, tokens[i], errs[i] = e.Join(context.Background(), fmt.Sprintf("P%d", i), "")
		}()
	}
	wg.Wait()

	seen := map[string]bool{}
	for i, err := range errs {
		if err != nil {
			t.Fatalf("join %d: %v", i, err)
		}
		if seen[tokens[i]] {
			t.Fatalf("duplicate token %q", tokens[i])
		}
		seen[tokens[i]] = true
	}
}

func TestJoinResumesByToken(t *testing.T) {
	e := startEngine(t, time.Hour)
	ctx := context.Background()
	pid, token, err := e.Join(ctx, "Alice", "")
	if err != nil {
		t.Fatal(err)
	}
	again, sameToken, err := e.Join(ctx, "", token)
	if err != nil || again != pid || sameToken != token {
		t.Fatalf("resume: id=%s token=%q err=%v, want %s", again, sameToken, err, pid)
	}
	if _, _, err := e.Join(ctx, "", "bogus"); err != ErrUnknownToken {
		t.Fatalf("unknown token without name: %v", err)
	}
}

func TestJoinReclaimsOfflineName(t *testing.T) {
	e := startEngine(t, time.Hour)
	ctx := context.Background()

	// Joined but never connected (e.g. the welcome got lost): the name is free to reclaim.
	lost, lostToken, err := e.Join(ctx, "Nova", "")
	if err != nil {
		t.Fatal(err)
	}
	pid, token, err := e.Join(ctx, "nova", "")
	if err != nil || pid != lost || token == lostToken {
		t.Fatalf("reclaim never-connected player: id=%s err=%v, want %s with a new token", pid, err, lost)
	}

	// Connected: the name is taken, also with an outdated token.
	sub, err := e.Subscribe(ctx, pid)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := e.Join(ctx, "Nova", "outdated"); err != game.ErrNameTaken {
		t.Fatalf("name of a connected player: %v, want %v", err, game.ErrNameTaken)
	}

	// After Leave / disconnect: the same player comes back, old tokens still work.
	e.Unsubscribe(sub)
	again, newToken, err := e.Join(ctx, "Nova", "")
	if err != nil || again != pid {
		t.Fatalf("rejoin after disconnect: id=%s err=%v, want %s", again, err, pid)
	}
	for _, tok := range []string{lostToken, token, newToken} {
		if got, _, err := e.Join(ctx, "", tok); err != nil || got != pid {
			t.Fatalf("token %q: id=%s err=%v, want %s", tok, got, err, pid)
		}
	}

	// Reclaiming doesn't create players or ships.
	var players, ships int
	e.exec(ctx, func() (bool, error) {
		players, ships = len(e.world.Players), len(e.world.Ships)
		return false, nil
	})
	if players != 1 || ships != 1 {
		t.Fatalf("%d players, %d ships after reclaiming, want 1 and 1", players, ships)
	}
}
