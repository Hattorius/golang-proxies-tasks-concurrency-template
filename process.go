package main

import (
	"log/slog"
	"time"
)

func process(task Task, proxy string) error {
	slog.Info("Processing", "proxy", proxy, "task", task.Data)
	time.Sleep(time.Second)
	return nil
}
