package domain

import (
	"fmt"
	"time"
)

// Experiment represents the complete chaos engineering experiment configuration
type Experiment struct {
	Name        string     `yaml:"name"`
	Description string     `yaml:"description"`
	SteadyState SteadyState `yaml:"steady_state"`
	Scenario    Scenario   `yaml:"scenario"`
}

// SteadyState defines how to validate the system is in a healthy state
type SteadyState struct {
	Probes []Probe `yaml:"probes"`
}

// Probe defines a health check to validate steady state
type Probe struct {
	Name    string        `yaml:"name"`
	Type    string        `yaml:"type"`
	HTTP    *HTTPProbe   `yaml:"http,omitempty"`
	Timeout time.Duration `yaml:"timeout"`
}

// HTTPProbe defines HTTP-specific probe configuration
type HTTPProbe struct {
	URL            string            `yaml:"url"`
	Method         string            `yaml:"method"`
	ExpectedStatus int               `yaml:"expected_status"`
	Headers        map[string]string `yaml:"headers,omitempty"`
}

// Scenario defines the chaos scenario to execute
type Scenario struct {
	Type     string        `yaml:"type"`
	Selector Selector      `yaml:"selector"`
	Duration time.Duration `yaml:"duration"`
	Interval time.Duration `yaml:"interval"`
}

// Selector defines how to select targets for chaos
type Selector struct {
	App string `yaml:"app"`
}

// Validate checks if the experiment configuration is valid
func (e *Experiment) Validate() error {
	if e.Name == "" {
		return fmt.Errorf("experiment name is required")
	}
	
	if len(e.SteadyState.Probes) == 0 {
		return fmt.Errorf("at least one probe is required")
	}
	
	for i, probe := range e.SteadyState.Probes {
		if err := probe.Validate(); err != nil {
			return fmt.Errorf("probe %d: %w", i, err)
		}
	}
	
	if err := e.Scenario.Validate(); err != nil {
		return fmt.Errorf("scenario: %w", err)
	}
	
	return nil
}

// Validate checks if the probe configuration is valid
func (p *Probe) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("probe name is required")
	}
	
	if p.Type != "http" {
		return fmt.Errorf("only 'http' probe type is supported")
	}
	
	if p.HTTP == nil {
		return fmt.Errorf("http configuration is required for http probes")
	}
	
	if p.HTTP.URL == "" {
		return fmt.Errorf("http url is required")
	}
	
	if p.HTTP.Method == "" {
		p.HTTP.Method = "GET"
	}
	
	if p.HTTP.ExpectedStatus == 0 {
		p.HTTP.ExpectedStatus = 200
	}
	
	if p.Timeout == 0 {
		p.Timeout = 30 * time.Second
	}
	
	return nil
}

// Validate checks if the scenario configuration is valid
func (s *Scenario) Validate() error {
	if s.Type != "restart_random" {
		return fmt.Errorf("only 'restart_random' scenario type is supported")
	}
	
	if s.Selector.App == "" {
		return fmt.Errorf("app selector is required")
	}
	
	if s.Duration <= 0 {
		return fmt.Errorf("positive duration is required")
	}
	
	if s.Interval <= 0 {
		return fmt.Errorf("positive interval is required")
	}
	
	if s.Interval > s.Duration {
		return fmt.Errorf("interval cannot be greater than duration")
	}
	
	return nil
}
