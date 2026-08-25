package scheduler

import (
	"context"
	"log"
	"time"
)

type Runner struct {
	Interval time.Duration
	Job      func(context.Context) error
}

func (r Runner) Start(ctx context.Context) {
	if r.Interval <= 0 || r.Job == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(r.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := r.Job(ctx); err != nil {
					log.Printf("scheduled scan failed: %v", err)
				}
			}
		}
	}()
}
