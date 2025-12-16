package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gimli-chaos/gimli/domain"
	"github.com/gimli-chaos/gimli/runner"
	"github.com/gimli-chaos/gimli/saboteur"
	"gopkg.in/yaml.v3"
)

const (
	appName    = "gimli"
	appVersion = "0.1.0"
)

func main() {
	// Parse command line arguments
	var (
		configFile = flag.String("config", "experiment.yaml", "Path to experiment configuration file")
		version    = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	if *version {
		fmt.Printf("%s version %s\n", appName, appVersion)
		os.Exit(0)
	}

	if *configFile == "" {
		log.Fatal("❌ Configuration file is required")
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("🛑 Received shutdown signal, cancelling experiment...")
		cancel()
	}()

	// Load and parse configuration
	log.Printf("📖 Loading configuration from %s", *configFile)
	experiment, err := loadExperiment(*configFile)
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	log.Printf("✅ Configuration loaded: %s", experiment.Name)

	// Create saboteur
	sab, err := saboteur.NewFlySaboteur()
	if err != nil {
		log.Fatalf("❌ Failed to create saboteur: %v", err)
	}

	// Create runner
	runner := runner.NewRunner(sab, log.Default())

	// Run experiment
	log.Println("🚀 Starting chaos engineering experiment...")
	startTime := time.Now()

	if err := runner.RunExperiment(ctx, experiment); err != nil {
		duration := time.Since(startTime)
		log.Fatalf("❌ Experiment failed after %v: %v", duration, err)
	}

	duration := time.Since(startTime)
	log.Printf("🎉 Experiment completed successfully in %v", duration)
}

// loadExperiment reads and parses the experiment configuration
func loadExperiment(filename string) (*domain.Experiment, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var experiment domain.Experiment
	if err := yaml.Unmarshal(data, &experiment); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	// Validate the configuration
	if err := experiment.Validate(); err != nil {
		return nil, fmt.Errorf("validating configuration: %w", err)
	}

	return &experiment, nil
}
