package client

import (
	"context"
	"time"
)

// RateLimiter steuert die Frequenz von ausgehenden Requests.
type RateLimiter struct {
	tokens chan struct{}
	ticker *time.Ticker
	done   chan struct{}
}

// NewRateLimiter erstellt einen neuen Limiter mit der angegebenen Anzahl an Requests pro Sekunde (rps).
func NewRateLimiter(rps int) *RateLimiter {
	if rps <= 0 {
		rps = 10
	}

	// Puffergröße entspricht rps, um kurze Spikes abzufangen
	tokens := make(chan struct{}, rps)

	// Tokens initial auffüllen
	for i := 0; i < rps; i++ {
		tokens <- struct{}{}
	}

	interval := time.Second / time.Duration(rps)
	ticker := time.NewTicker(interval)
	done := make(chan struct{})

	rl := &RateLimiter{
		tokens: tokens,
		ticker: ticker,
		done:   done,
	}

	// Hintergrund-Routine gießt kontinuierlich neue Tokens nach
	go func() {
		for {
			select {
			case <-ticker.C:
				select {
				case tokens <- struct{}{}:
				default:
					// Puffer ist voll, Token verfällt
				}
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()

	return rl
}

// Wait blockiert, bis ein Token verfügbar ist oder der Context abgebrochen wird.
func (rl *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-rl.tokens:
		return nil
	}
}

// Stop beendet den internen Ticker.
func (rl *RateLimiter) Stop() {
	close(rl.done)
}
