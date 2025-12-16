package runner

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gimli-chaos/gimli/domain"
	"github.com/gimli-chaos/gimli/saboteur"
)

// Runner orchestrates chaos engineering experiments
type Runner struct {
	saboteur saboteur.Saboteur
	logger   *log.Logger
}

// NewRunner creates a new experiment runner
func NewRunner(saboteur saboteur.Saboteur, logger *log.Logger) *Runner {
	if logger == nil {
		logger = log.Default()
	}
	
	return &Runner{
		saboteur: saboteur,
		logger:   logger,
	}
}

// RunExperiment executes the complete chaos engineering experiment
func (r *Runner) RunExperiment(ctx context.Context, experiment *domain.Experiment) error {
	r.logger.Printf("🚀 Starting experiment: %s", experiment.Name)
	r.logger.Printf("📋 Description: %s", experiment.Description)
	
	// Validate steady state before starting
	r.logger.Println("🔍 Validating steady state...")
	if err := r.RunProbes(ctx, experiment.SteadyState.Probes); err != nil {
		return fmt.Errorf("steady state validation failed: %w", err)
	}
	r.logger.Println("✅ Steady state validated")
	
	// Execute the chaos scenario
	if err := r.executeScenario(ctx, experiment); err != nil {
		return fmt.Errorf("scenario execution failed: %w", err)
	}
	
	// Validate steady state after chaos
	r.logger.Println("🔍 Validating steady state after chaos...")
	if err := r.RunProbes(ctx, experiment.SteadyState.Probes); err != nil {
		return fmt.Errorf("steady state validation failed after chaos: %w", err)
	}
	r.logger.Println("✅ Steady state maintained after chaos")
	
	r.logger.Println("🎉 Experiment completed successfully")
	return nil
}

// RunProbes executes health checks to validate steady state
func (r *Runner) RunProbes(ctx context.Context, probes []domain.Probe) error {
	var wg sync.WaitGroup
	errors := make(chan error, len(probes))
	
	for _, probe := range probes {
		wg.Add(1)
		go func(p domain.Probe) {
			defer wg.Done()
			
			if err := r.executeProbe(ctx, p); err != nil {
				errors <- fmt.Errorf("probe '%s' failed: %w", p.Name, err)
			} else {
				r.logger.Printf("✅ Probe '%s' passed", p.Name)
			}
		}(probe)
	}
	
	wg.Wait()
	close(errors)
	
	// Collect any errors
	var probeErrors []error
	for err := range errors {
		probeErrors = append(probeErrors, err)
	}
	
	if len(probeErrors) > 0 {
		return fmt.Errorf("probes failed: %v", probeErrors)
	}
	
	return nil
}

// executeProbe runs a single probe
func (r *Runner) executeProbe(ctx context.Context, probe domain.Probe) error {
	if probe.Type != "http" {
		return fmt.Errorf("unsupported probe type: %s", probe.Type)
	}
	
	httpProbe := probe.HTTP
	if httpProbe == nil {
		return fmt.Errorf("HTTP probe configuration is missing")
	}
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, httpProbe.Method, httpProbe.URL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	
	// Add headers
	for key, value := range httpProbe.Headers {
		req.Header.Set(key, value)
	}
	
	// Execute request with timeout
	client := &http.Client{
		Timeout: probe.Timeout,
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()
	
	// Consume response body to ensure connection reuse
	io.Copy(io.Discard, resp.Body)
	
	// Check status code
	if resp.StatusCode != httpProbe.ExpectedStatus {
		return fmt.Errorf("unexpected status code: got %d, want %d", resp.StatusCode, httpProbe.ExpectedStatus)
	}
	
	return nil
}

// executeScenario runs the chaos scenario
func (r *Runner) executeScenario(ctx context.Context, experiment *domain.Experiment) error {
	scenario := experiment.Scenario
	r.logger.Printf("🔥 Executing chaos scenario: %s for %v (interval: %v)",
		scenario.Type, scenario.Duration, scenario.Interval)
	
	// Get targets for chaos
	targets, err := r.saboteur.ListTargets(ctx, scenario.Selector.App, nil)
	if err != nil {
		return fmt.Errorf("listing targets: %w", err)
	}
	
	r.logger.Printf("🎯 Found %d eligible targets", len(targets))
	
	// Create ticker for interval-based chaos
	ticker := time.NewTicker(scenario.Interval)
	defer ticker.Stop()
	
	// Create timeout for total duration
	durationCtx, cancel := context.WithTimeout(ctx, scenario.Duration)
	defer cancel()
	
	attackCount := 0
	
	for {
		select {
		case <-durationCtx.Done():
			r.logger.Printf("⏰ Scenario duration completed. Attacks executed: %d", attackCount)
			return nil
			
		case <-ticker.C:
			// Execute attack
			if err := r.executeAttack(durationCtx, scenario, targets); err != nil {
				r.logger.Printf("⚠️  Attack failed: %v", err)
				continue
			}
			attackCount++
			
			// Validate steady state after attack
			r.logger.Println("🔍 Validating steady state after attack...")
			if err := r.RunProbes(durationCtx, experiment.SteadyState.Probes); err != nil {
				return fmt.Errorf("steady state lost after attack: %w", err)
			}
		}
	}
}

// executeAttack performs a single chaos attack
func (r *Runner) executeAttack(ctx context.Context, scenario domain.Scenario, targets []saboteur.Target) error {
	if scenario.Type != "restart_random" {
		return fmt.Errorf("unsupported scenario type: %s", scenario.Type)
	}
	
	// Select random target
	if len(targets) == 0 {
		return fmt.Errorf("no targets available")
	}
	
	target := targets[rand.Intn(len(targets))]
	
	r.logger.Printf("💥 Attacking machine %s (%s)...", target.ID, target.Name)
	
	if err := r.saboteur.RestartMachine(ctx, scenario.Selector.App, target.ID); err != nil {
		return fmt.Errorf("restarting machine %s: %w", target.ID, err)
	}
	
	r.logger.Printf("✅ Successfully attacked machine %s", target.ID)
	
	// Wait a moment for the restart to take effect
	time.Sleep(2 * time.Second)
	
	return nil
}
