# Gimli - Chaos Engineering Runner for Fly.io

Gimli is a powerful Chaos Engineering tool designed specifically for testing the resilience of applications deployed on Fly.io. It reads experiment configurations from YAML files, validates system health through HTTP probes, and executes controlled chaos scenarios by restarting random Fly.io machines.

## Features

- **YAML-based Configuration**: Define experiments using simple, human-readable YAML files
- **HTTP Health Probes**: Validate system steady state before, during, and after chaos
- **Scheduled Chaos Scenarios**: Execute attacks at regular intervals over a specified duration
- **Fly.io Integration**: Direct integration with Fly Machines API for machine management
- **Context-based Cancellation**: Graceful shutdown with Ctrl+C support
- **Comprehensive Logging**: Detailed logging of all experiment phases and actions

## Installation

```bash
go build -o gimli .
```

## Usage

### Basic Usage
```bash
# Run with default configuration file (experiment.yaml)
./gimli

# Run with custom configuration file
./gimli -config my-experiment.yaml

# Show version
./gimli -version

# Show help
./gimli -help
```

### Environment Variables
- `FLY_API_TOKEN`: Required for authenticating with Fly.io API

## Configuration

### Example experiment.yaml
```yaml
name: "fly-io-app-restart-test"
description: "Test application resilience by randomly restarting Fly.io machines"

steady_state:
  probes:
    - name: "health-check"
      type: "http"
      timeout: 30s
      http:
        url: "https://my-app.fly.dev/health"
        method: "GET"
        expected_status: 200
        headers:
          User-Agent: "gimli-chaos-probe"
    
    - name: "api-endpoint"
      type: "http"
      timeout: 15s
      http:
        url: "https://my-app.fly.dev/api/status"
        method: "GET"
        expected_status: 200

scenario:
  type: "restart_random"
  selector:
    app: "my-app"
  duration: 5m
  interval: 30s
```

## Architecture

### Core Components

1. **domain/models.go**: Defines the data structures for experiments, probes, and scenarios
2. **saboteur/fly_saboteur.go**: Implements Fly.io API interactions
3. **runner/runner.go**: Contains the main experiment logic and probe execution
4. **main.go**: Orchestrates the entire experiment flow

### Experiment Flow

1. **Configuration Loading**: Parse and validate YAML configuration
2. **Steady State Validation**: Run all defined probes to ensure system is healthy
3. **Chaos Execution**: Execute attacks at specified intervals
4. **Continuous Validation**: Verify steady state after each attack
5. **Completion**: Final steady state validation and cleanup

## Supported Features

### Probe Types
- `http`: HTTP health checks with configurable method, headers, and expected status

### Scenario Types
- `restart_random`: Restart random Fly.io machines at specified intervals

## Error Handling

The tool implements comprehensive error handling:
- Configuration validation with detailed error messages
- HTTP probe timeout and retry logic
- Graceful handling of Fly.io API errors
- Context-based cancellation for clean shutdowns

## Security

- API tokens are read from environment variables (never hardcoded)
- Secure HTTP client configuration
- No sensitive data logged to console

## Development

### Building
```bash
go build -o gimli .
```

### Testing
```bash
# Test compilation
go build .

# Run with example configuration
export FLY_API_TOKEN="your-token-here"
./gimli -config experiment.yaml
```

## License

This project is licensed under the MIT License.
