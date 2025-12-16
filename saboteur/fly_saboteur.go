package saboteur

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Saboteur interface defines the contract for chaos operations
type Saboteur interface {
	ListTargets(ctx context.Context, app string, selector map[string]string) ([]Target, error)
	RestartMachine(ctx context.Context, app string, machineID string) error
}

// Target represents a Fly.io machine that can be targeted for chaos
type Target struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	State    string            `json:"state"`
	Region   string            `json:"region"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// FlySaboteur implements the Saboteur interface for Fly.io
type FlySaboteur struct {
	apiToken   string
	httpClient *http.Client
	baseURL    string
}

// NewFlySaboteur creates a new Fly.io saboteur instance
func NewFlySaboteur() (*FlySaboteur, error) {
	apiToken := os.Getenv("FLY_API_TOKEN")
	if apiToken == "" {
		return nil, fmt.Errorf("FLY_API_TOKEN environment variable is required")
	}
	
	return &FlySaboteur{
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.machines.dev/v1",
	}, nil
}

// ListTargets fetches and filters machines from Fly.io
func (f *FlySaboteur) ListTargets(ctx context.Context, app string, selector map[string]string) ([]Target, error) {
	url := fmt.Sprintf("%s/apps/%s/machines", f.baseURL, app)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", f.apiToken))
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}
	
	var machines []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		State string `json:"state"`
		Region string `json:"region"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&machines); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	
	// Convert to targets and filter by selector
	var targets []Target
	for _, machine := range machines {
		// Only include machines that are running or started
		if machine.State != "started" && machine.State != "running" {
			continue
		}
		
		target := Target{
			ID:     machine.ID,
			Name:   machine.Name,
			State:  machine.State,
			Region: machine.Region,
		}
		
		// Apply selector filters (simplified - in real implementation would match labels)
		targets = append(targets, target)
	}
	
	if len(targets) == 0 {
		return nil, fmt.Errorf("no eligible targets found in app %s", app)
	}
	
	return targets, nil
}

// RestartMachine restarts a specific Fly.io machine
func (f *FlySaboteur) RestartMachine(ctx context.Context, app string, machineID string) error {
	url := fmt.Sprintf("%s/apps/%s/machines/%s/restart", f.baseURL, app, machineID)
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", f.apiToken))
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}
