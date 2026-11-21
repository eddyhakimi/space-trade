package engine

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestSlowSubscriberIsDropped(t *testing.T) {
	e := startEngine(t, time.Hour)
	_, slow := joinAndSubscribe(t, e, "Slow")
	_, fast := joinAndSubscribe(t, e, "Fast")

	const joins = 3 * subBuffer
	var received int
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range fast.C {
			if received++; received == joins+1 { // +1: Fast's own subscribe
				return
			}
		}
	}()

	for i := range joins {
		if _, _, err := e.Join(context.Background(), fmt.Sprintf("P%d", i), ""); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("fast subscriber got only %d messages", received)
	}

	n := 0
	for range slow.C { // must terminate: channel closed after the buffer filled up
		n++
	}
	if n != subBuffer {
		t.Fatalf("slow subscriber got %d messages before being dropped, want %d", n, subBuffer)
	}
}
