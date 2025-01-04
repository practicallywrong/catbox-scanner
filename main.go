package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"catbox-scanner/internals/config"
	"catbox-scanner/internals/database"
	"catbox-scanner/internals/metrics"
	"catbox-scanner/internals/scanner"

	"github.com/panjf2000/ants/v2"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	metricss := metrics.NewMetrics(60)
	isRunning := true

	db, err := database.NewDatabase(cfg.Database.Filepath)
	if err != nil {
		fmt.Printf("Failed to initialize database: %v\n", err)
		return
	}
	defer db.Close()

	pool, _ := ants.NewPool(
		cfg.Scanner.NumWorkers,
		ants.WithNonblocking(false),
		ants.WithDisablePurge(true),
		ants.WithPreAlloc(true),
	)

	scannerService := scanner.NewScanner(cfg, metricss, db, pool, &isRunning)

	go metricss.StartPrintLoop()
	go metrics.StartPrometheusServer()
	go scannerService.StartScanning()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan
	fmt.Println("\nShutting down gracefully...")
	isRunning = false
	pool.ReleaseTimeout(time.Second)
	db.Close()
	fmt.Println("Bye Bye~")
}
