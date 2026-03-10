package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fenco/trademate/services/api/internal/config"
	"github.com/fenco/trademate/services/api/internal/store"
	"github.com/fenco/trademate/services/api/internal/worker"
)

func main() {
	mode := flag.String("mode", "once", "worker mode: once or loop")
	interval := flag.Duration("interval", 30*time.Second, "poll interval when mode=loop")
	storeID := flag.String("store-id", "", "optional store filter")
	limit := flag.Int("limit", 20, "max queued tasks to process per run")
	flag.Parse()

	cfg := config.Load()
	db, err := store.OpenDB(cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("failed to connect mysql: %v", err)
	}
	defer db.Close()

	wd, _ := os.Getwd()
	if err := store.ApplyMigrations(db, wd); err != nil {
		log.Fatalf("failed to apply migrations: %v", err)
	}

	repo := store.NewRepository(db)
	svc := worker.NewService(repo, nil)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	run := func() {
		result, runErr := svc.RunOnce(ctx, worker.RunOnceInput{
			StoreID: *storeID,
			Limit:   *limit,
			ActorID: "system_worker",
		})
		if runErr != nil {
			log.Printf("worker run failed: %v", runErr)
			return
		}

		log.Printf("worker run finished: picked=%d succeeded=%d failed=%d skipped=%d", result.Picked, result.Succeeded, result.Failed, result.Skipped)
		for _, item := range result.Results {
			log.Printf("task=%s store=%s type=%s status=%s msg=%s", item.TaskID, item.StoreID, item.TaskType, item.Status, item.Message)
		}
	}

	switch *mode {
	case "once":
		run()
	case "loop":
		ticker := time.NewTicker(*interval)
		defer ticker.Stop()

		run()
		for {
			select {
			case <-ctx.Done():
				log.Printf("worker stopped")
				return
			case <-ticker.C:
				run()
			}
		}
	default:
		log.Fatalf("invalid mode: %s", *mode)
	}
}
