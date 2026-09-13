// Package scheduler menyediakan runner tugas periodik dengan interval
// yang dapat dikonfigurasi. Setiap tugas dijalankan pada goroutine sendiri
// dan berhenti dengan rapi saat context dibatalkan.
package scheduler

import (
	"context"
	"log/slog"
	"time"
)

type Job struct {
	Name     string
	Interval time.Duration
	Timeout  time.Duration
	Run      func(ctx context.Context) error
}

func New(name string, interval, timeout time.Duration, run func(ctx context.Context) error) Job {
	if interval <= 0 {
		interval = time.Minute
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return Job{Name: name, Interval: interval, Timeout: timeout, Run: run}
}

// Start menjalankan seluruh tugas sampai ctx dibatalkan.
// Tugas dimulai setelah satu interval agar segera berjalan maupun
// mengulang sesuai jadwal.
func Start(ctx context.Context, logger *slog.Logger, jobs ...Job) {
	for _, job := range jobs {
		job := job
		go func() {
			ticker := time.NewTicker(job.Interval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					taskCtx, cancel := context.WithTimeout(ctx, job.Timeout)
					err := job.Run(taskCtx)
					cancel()
					if err != nil && logger != nil {
						logger.Error("scheduled job failed", "job", job.Name, "error", err, "interval", job.Interval.String())
					}
				}
			}
		}()
	}
	<-ctx.Done()
}
