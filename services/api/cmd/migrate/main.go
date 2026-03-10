package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/fenco/trademate/services/api/internal/config"
	"github.com/fenco/trademate/services/api/internal/store"
)

func main() {
	action := flag.String("action", "up", "migration action: up|down|reset")
	seed := flag.Bool("seed", false, "seed demo data after migration")
	flag.Parse()

	cfg := config.Load()
	db, err := store.OpenDB(cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("connect mysql failed: %v", err)
	}
	defer db.Close()

	cwd, _ := os.Getwd()

	switch *action {
	case "up":
		must(store.ApplyMigrations(db, cwd))
		if *seed {
			must(store.SeedDemoData(db))
		}
		log.Println("migrate up success")
	case "down":
		must(store.RollbackMigrations(db))
		log.Println("migrate down success")
	case "reset":
		must(store.RollbackMigrations(db))
		must(store.ApplyMigrations(db, cwd))
		if *seed {
			must(store.SeedDemoData(db))
		}
		log.Println("migrate reset success")
	default:
		log.Fatalf("unsupported action: %s", *action)
	}
}

func must(err error) {
	if err == nil {
		return
	}

	fmt.Println(err)
	os.Exit(1)
}
