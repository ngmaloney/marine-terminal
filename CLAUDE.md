# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## ⚠️ CRITICAL: TEST-DRIVEN DEVELOPMENT REQUIREMENTS

**MANDATORY FOR ALL CODE CHANGES:**

1. **ALWAYS run tests after ANY code change**: `go test ./...`
2. **ALL tests MUST pass** before work is considered complete
3. **Fix failing tests immediately** - do not move on to other tasks
4. **Add tests for new functionality** before or alongside implementation
5. **Update tests when refactoring** to match new behavior

**Test Execution Protocol:**
```bash
# After EVERY change, run:
go test ./...

# If any tests fail:
# 1. Read the test failure output carefully
# 2. Fix the code OR update the test to match correct behavior
# 3. Re-run tests until ALL pass
# 4. Only then is the work complete
```

**No exceptions.** Passing tests are not optional - they are the definition of working code.

## Project Overview

Marine Terminal is a terminal application for displaying NOAA weather and tide information for US ports. The application retrieves and displays:
- Current weather and 3-day forecast
- NOAA wind predictions
- Wave heights
- NOAA alerts for the selected port
- Tide information for next 3 days

The interface uses colored pane views with optional UTF emoji for weather conditions. Users can search for ports by postal code or city/state.

## Technology Stack

- **Language**: Go
- **TUI Framework**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Built on The Elm Architecture
- **Complementary Libraries**:
  - [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components (viewports, lists, spinners, etc.)
  - [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling and layout
  - [Harmonica](https://github.com/charmbracelet/harmonica) - Animation utilities

## Project Structure

Follow standard Go project layout:
```
marine-terminal/
├── cmd/
│   └── marine-terminal/      # Main application entry point
│       └── main.go
├── internal/             # Private application code
│   ├── ui/              # Bubble Tea UI components and models
│   ├── noaa/            # NOAA API clients
│   ├── models/          # Data models (Weather, Tide, Alert, etc.)
│   └── ports/           # Port search and location logic
├── pkg/                 # Public libraries (if any)
├── testdata/            # Test fixtures and sample data
├── go.mod
├── go.sum
└── README.md
```

Use `internal/` for code that should not be imported by external projects. Keep packages focused and cohesive.

## Development Commands

```bash
# Initialize Go module (if not already done)
go mod init github.com/ngmaloney/marine-terminal

# Install dependencies
go mod download

# Build the application
go build -o marine-terminal ./cmd/marine-terminal

# Run the application
go run ./cmd/marine-terminal

# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run a specific test
go test -run TestName ./path/to/package

# Run tests with race detector
go test -race ./...

# Format code
go fmt ./...

# Lint (requires golangci-lint)
golangci-lint run

# Run tests and build before committing
go test ./... && go build ./cmd/marine-terminal
```

## Testing Requirements

**All code must include comprehensive tests.** Follow these practices:

### Test Coverage
- Write tests for all new functions and methods
- Target minimum 80% coverage for business logic
- Use `go test -cover ./...` to verify coverage

### Test Organization
- Place tests in `*_test.go` files in the same package
- Use `testdata/` directory for fixtures and sample responses
- Mock external dependencies (HTTP clients for NOAA APIs)

### Table-Driven Tests
Use Go's table-driven test pattern for multiple test cases:
```go
func TestParseWeatherData(t *testing.T) {
    tests := []struct {
        name    string
        input   []byte
        want    WeatherData
        wantErr bool
    }{
        {"valid data", validJSON, expectedData, false},
        {"invalid JSON", invalidJSON, WeatherData{}, true},
        {"empty response", []byte{}, WeatherData{}, true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseWeatherData(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseWeatherData() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("ParseWeatherData() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Testing Bubble Tea Components
- Test Update function with various message types
- Verify Model state transitions
- Test View rendering for different states (loading, error, success)
- Use testable interfaces for external dependencies

### Running Tests
**MANDATORY - Tests must pass after EVERY change:**
1. Run `go test ./...` after EVERY code change
2. **ALL tests must pass** - zero failures, zero exceptions
3. Fix failing tests immediately - do not proceed to other work
4. Add new tests for bug fixes to prevent regressions
5. Run with `-race` flag to detect race conditions
6. Never commit code with failing tests
7. Never leave test failures for "later"

**If tests fail:**
- Stop all other work
- Read the failure output
- Fix the code or update tests to match correct behavior
- Re-run tests until 100% pass
- Only then continue with other tasks

## Go Best Practices

### Code Style
- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Use `gofmt` to format all code
- Run `go vet` to catch common mistakes
- Use meaningful variable names (avoid single-letter names except for short-lived loop variables)

### Error Handling
- Always check and handle errors; never ignore them
- Wrap errors with context: `fmt.Errorf("failed to fetch weather data: %w", err)`
- Return errors rather than panicking in library code
- Log errors at appropriate levels in the UI layer

### Interfaces and Abstraction
- Define interfaces where they're used, not where they're implemented
- Keep interfaces small and focused (prefer many small interfaces)
- Use interfaces to make code testable (e.g., HTTP client interface for NOAA API)

Example:
```go
// internal/noaa/client.go
type WeatherClient interface {
    GetForecast(lat, lon float64) (*WeatherData, error)
    GetAlerts(stationID string) ([]Alert, error)
}

// Makes testing easier - can mock the interface
type MockWeatherClient struct {
    ForecastFunc func(lat, lon float64) (*WeatherData, error)
}
```

### Concurrency
- Use goroutines sparingly and only when needed
- Always consider goroutine lifecycle and cleanup
- Use context.Context for cancellation and timeouts
- Be mindful of race conditions (test with `-race`)

### Dependency Management
- Keep dependencies minimal
- Pin versions in `go.mod`
- Run `go mod tidy` regularly to clean up unused dependencies

## Data Sources

The application integrates with NOAA APIs to retrieve:
- Weather forecasts and current conditions
- Wind predictions
- Wave height data
- Marine alerts and warnings
- Tide predictions

When implementing API integrations, ensure proper error handling for network failures and API rate limits. NOAA provides multiple API endpoints (weather.gov API, CO-OPS for tides) that may need to be coordinated.

## Bubble Tea Architecture (The Elm Architecture)

Bubble Tea applications follow The Elm Architecture with three core components:

### Model
The Model struct holds the complete application state:
- Current port/location information
- Weather data, wind predictions, wave heights
- Tide schedules
- NOAA alerts
- UI state (active pane, loading states, error messages)
- Component models from Bubbles (e.g., viewport, list, spinner)

### Update
The Update function processes messages (Msgs) and returns an updated Model plus optional Commands:
- Handle user input (keypresses for navigation, search input)
- Process API responses (weather data fetched, tide data received)
- Manage async operations via Commands (tea.Cmd)
- Update component states (Bubbles components have their own Update methods)

### View
The View function renders the UI as a string based on the current Model:
- Use Lipgloss for styling panes and layout
- Compose multiple pane views (weather, tides, alerts, etc.)
- Leverage Bubbles components for common UI elements
- Return the complete rendered string

### Handling Async Data

API calls to NOAA services should be initiated via Commands returned from Update. Use `tea.Cmd` to perform I/O operations:
- Create custom message types for API responses
- Return commands that fetch data in the background
- Handle loading states in the Model
- Process results when messages arrive back in Update

Example pattern:
```go
type weatherDataMsg struct { data WeatherData; err error }

func fetchWeatherData(port string) tea.Cmd {
    return func() tea.Msg {
        data, err := callNOAAAPI(port)
        return weatherDataMsg{data, err}
    }
}
```

### Multi-Pane Layout

For the pane-based view, consider:
- Using Lipgloss layout functions (JoinHorizontal, JoinVertical) to compose panes
- Maintaining separate Model fields for each pane's state
- Implementing keyboard shortcuts to switch active panes
- Using Bubbles viewport component for scrollable content within panes

## Port Search Implementation

Port search by postal code or city/state requires either:
- A lookup database/mapping of locations to NOAA station IDs
- Integration with NOAA's station search capabilities
- Geocoding to map user input to nearby marine stations

The implementation should handle ambiguous searches (e.g., multiple ports near a city) and provide clear selection mechanisms.

## Data Refresh Strategy

Consider how frequently to refresh different data types:
- Current conditions may update more frequently
- Tide predictions are typically static once fetched
- Alerts should be checked regularly for safety-critical information

The UI should indicate data freshness and loading states clearly.
