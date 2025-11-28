# Mariner TUI

A terminal-based application for displaying NOAA weather and tide information for US marine ports.

## Features

- **Marine Weather Conditions**: Current conditions and 3-day forecasts
- **NOAA Wind Predictions**: Wind speed, direction, and gusts in knots
- **Wave Heights**: Detailed wave/swell information with direction and period
- **Tide Predictions**: High and low tides for the next 3 days
- **NOAA Marine Alerts**: Small craft advisories, gale warnings, and other marine alerts
- **Real-Time Station Search**: Access to 3,379+ NOAA tide stations via live API
- **Smart Caching**: Fetches station database once, then searches instantly
- **Port Search**: Search by city name or state abbreviation
- **Multi-Pane Interface**: Color-coded panes with keyboard navigation

## Installation

### Prerequisites

- Go 1.21 or later
- (Optional) [Task](https://taskfile.dev) - Modern task runner for development

### Build from Source

```bash
# Clone the repository
git clone https://github.com/ngmaloney/mariner-tui.git
cd mariner-tui

# Download dependencies
GO111MODULE=on go mod tidy

# Build the application
GO111MODULE=on go build -o mariner-tui ./cmd/mariner-tui

# Run the application
./mariner-tui
```

**Note**: If you're using an older version of Go, you may need to set `GO111MODULE=on` to enable module support.

### First Run: Automatic Data Provisioning

The application automatically downloads and configures marine zone data on first run:

```bash
./mariner-tui
```

**What happens on first run:**
1. Downloads NOAA marine zones shapefile (~12 MB)
2. Downloads zipcode CSV data (~42,000 zipcodes)
3. Builds SQLite database with 566+ marine forecast zones and zipcode data
4. Creates `data/mariner.db` (~32 MB)
5. Cleans up temporary files

**Total time:** ~1-2 seconds

**Subsequent runs:** Instant - uses existing database

The marine zones database is **not** included in the repository and will be downloaded automatically when needed. No manual setup required!

### Manual Data Provisioning

The database provisions automatically, but you can verify or rebuild it:

```bash
# Remove database to trigger re-provisioning
rm -rf data/

# Run any application that uses zones
GO111MODULE=on go run test_nearby_zones.go
```

The provisioning will:
- Download from: `https://www.weather.gov/source/gis/Shapefiles/WSOM/mz18mr25.zip`
- Extract marine zone boundaries (shapefiles)
- Build indexed SQLite database
- Store zone centers for distance calculations

## Usage

### Quick Start Guide

1. **Start the application**:
   ```bash
   ./mariner-tui
   ```

2. **Search for a port**: Type a city name, state abbreviation, or ZIP code
   - Try: `Chatham`, `02633`, `Woods Hole`, `MA`, `Seattle`, `98101`
   - First search loads ~3,300 stations (takes a few seconds)
   - Subsequent searches are instant

3. **View the data**: Press Enter to see weather, tides, and alerts

4. **Navigate**: Use Tab to switch between panes

5. **Search again**: Press `S` to find a different port

### Keyboard Navigation

**In Search Mode:**
- **Type**: Enter city name or state (e.g., "Seattle", "CA", "New York")
- **Backspace**: Delete characters
- **Enter**: Search for the port
- **Ctrl+C**: Quit the application

**In Display Mode:**
- **Tab**: Switch between panes (Weather → Tides → Alerts)
- **Shift+Tab**: Switch between panes in reverse
- **S**: Return to search to find a different port
- **Q** or **Ctrl+C**: Quit the application

## Marine Data Formats

The application displays marine conditions in NOAA format:

### Wind
```
W 15-20 kt, gusts 30 kt
```
- Direction (W = West)
- Speed range in knots
- Gust speed if applicable

### Seas
```
Seas 5-7 ft
Wave Detail:
  S 5 ft at 8 seconds
  W 4 ft at 5 seconds
```
- Overall sea height range
- Individual wave components with:
  - Direction (S = South swell, W = West swell)
  - Height in feet
  - Period in seconds

### Tides
```
Today Nov 27
  2:15 PM   Low   0.5 ft
  8:45 PM   High  5.2 ft
```
- Date and time
- Tide type (High/Low)
- Height in feet (MLLW datum)

## Station Coverage

The application provides access to **all active NOAA tide prediction stations** via the NOAA Metadata API:
- **3,379+ stations** across the United States
- Includes all coastal states: MA, CA, WA, NY, FL, TX, OR, NC, AK, HI, and more
- Examples: Chatham, Woods Hole, Seattle, San Francisco, Boston, New York
- Search by city name (e.g., "Chatham", "Seattle"), state (e.g., "MA", "CA"), or ZIP code (e.g., "02633", "98101")

## Development

### Using Task (Recommended)

This project uses [Task](https://taskfile.dev) for common development tasks. Task is a modern alternative to Make, written in Go.

**Install Task:**
```bash
# macOS
brew install go-task/tap/go-task

# Or using Go
go install github.com/go-task/task/v3/cmd/task@latest
```

**Common Commands:**
```bash
# Show all available tasks
task --list

# Build the application
task build

# Run the application
task run

# Run tests
task test

# Run tests with coverage
task test:coverage

# Format and lint code
task dev

# Check data provisioning status
task data:check

# Provision marine zones database
task data:provision

# Clean all generated files
task clean:all

# Show project info
task info
```

**Quick Development Workflow:**
```bash
# Format, lint, and test in one command
task dev

# Build and run
task build run

# Full CI check (lint + coverage + race detection)
task ci
```

### Manual Build (Without Task)

If you prefer not to use Task, you can use standard Go commands:

```bash
# Build
GO111MODULE=on go build -o mariner-tui ./cmd/mariner-tui

# Run
GO111MODULE=on go run ./cmd/mariner-tui

# Test
GO111MODULE=on go test ./...
```

### Project Structure

```
mariner-tui/
├── cmd/
│   └── mariner-tui/      # Main application
├── internal/
│   ├── models/           # Data models (Weather, Tide, Alert, Port)
│   ├── noaa/             # NOAA API clients
│   ├── ports/            # Port search and marine zone integration
│   ├── zonelookup/       # Marine zones database and provisioning
│   └── ui/               # Bubble Tea UI components
├── data/                 # Auto-generated (excluded from git)
│   └── mariner.db        # SQLite database with marine zones and zipcodes
├── testdata/             # Test fixtures
└── CLAUDE.md             # Development guide for Claude Code
```

### Running the Test Suite

**Run all tests:**
```bash
GO111MODULE=on go test ./...
```

**Run tests with verbose output:**
```bash
GO111MODULE=on go test ./... -v
```

**Run tests with coverage:**
```bash
GO111MODULE=on go test ./... -cover
```

**Run tests for a specific package:**
```bash
# Test only models
GO111MODULE=on go test ./internal/models/... -v

# Test only NOAA API clients
GO111MODULE=on go test ./internal/noaa/... -v

# Test only UI components
GO111MODULE=on go test ./internal/ui/... -v

# Test only port search
GO111MODULE=on go test ./internal/ports/... -v
```

**Run integration tests:**
```bash
GO111MODULE=on go test ./internal/ui/... -v -run TestIntegration
```

**Generate detailed coverage report:**
```bash
# Generate coverage file
GO111MODULE=on go test ./... -coverprofile=coverage.out

# View coverage in terminal
GO111MODULE=on go tool cover -func=coverage.out

# Generate HTML coverage report
GO111MODULE=on go tool cover -html=coverage.out
```

**Run tests with race detector:**
```bash
GO111MODULE=on go test ./... -race
```

**Run a specific test:**
```bash
# Run only the search and fetch integration test
GO111MODULE=on go test ./internal/ui/... -v -run TestIntegration_SearchAndFetchData

# Run only error handling tests
GO111MODULE=on go test ./internal/ui/... -v -run TestIntegration_ErrorHandling
```

### Expected Test Results

When you run `go test ./... -cover`, you should see:

```
ok      github.com/ngmaloney/mariner-tui/internal/models    coverage: 100.0%
ok      github.com/ngmaloney/mariner-tui/internal/noaa      coverage: 84.8%
ok      github.com/ngmaloney/mariner-tui/internal/ports     coverage: 100.0%
ok      github.com/ngmaloney/mariner-tui/internal/ui        coverage: 42.7%
```

**Total: 88 test cases, all passing**

### Test Coverage Breakdown

- **models**: 100.0% - Data structures for weather, tides, alerts, ports
- **ports**: 100.0% - Port search functionality
- **noaa**: 84.8% - NOAA API clients (weather, tides, alerts)
- **ui**: 42.7% - UI components and integration tests

## Data Sources

- **Stations**: [NOAA CO-OPS Metadata API](https://api.tidesandcurrents.noaa.gov/mdapi/prod/)
- **Marine Zones**: [NOAA Marine Zones Shapefile](https://www.weather.gov/gis/MarineZones) (auto-downloaded)
- **Weather**: [NOAA Weather API](https://www.weather.gov/documentation/services-web-api)
- **Tides**: [NOAA CO-OPS API](https://tidesandcurrents.noaa.gov/api/)
- **Alerts**: [NOAA Weather Alerts API](https://www.weather.gov/documentation/services-web-api)

### Marine Zones Database

The application uses NOAA's official marine forecast zone boundaries to determine which zones are near a given location. The marine zones database:

- **Auto-provisions** on first run (no manual setup)
- **566+ zones** covering all US coastal waters
- **42,000+ zipcodes** for fast location lookup
- **Distance-based lookup** - finds zones within configurable radius
- **Sorted by proximity** - shows nearest zones first
- **No static data** - all data from official NOAA shapefiles and CSV sources
- **Stored locally** in `data/mariner.db` (excluded from git)

## Technologies

- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)**: Terminal UI framework (Elm Architecture)
- **[Lipgloss](https://github.com/charmbracelet/lipgloss)**: Styling and layout
- **[Bubbles](https://github.com/charmbracelet/bubbles)**: TUI components
- **[modernc.org/sqlite](https://gitlab.com/cznic/sqlite)**: Pure Go SQLite (no CGO required)
- **[go-shp](https://github.com/jonas-p/go-shp)**: Shapefile reader for GIS data

## Contributing

Contributions are welcome! Please ensure:
- All tests pass: `go test ./...`
- Code is formatted: `go fmt ./...`
- New features include tests (aim for 80%+ coverage)
- Follow Go best practices

See [CLAUDE.md](CLAUDE.md) for detailed development guidelines.

## License

This project is provided as-is for educational and personal use.

## Troubleshooting

### Build Errors

**Error: "cannot find package"**

If you see errors like `cannot find package "github.com/charmbracelet/bubbletea"`, run:

```bash
GO111MODULE=on go mod download
GO111MODULE=on go mod tidy
```

Then try building again.

**Error: "modules disabled by GO111MODULE=off"**

Set the environment variable:
```bash
export GO111MODULE=on
```

Or prefix all go commands with `GO111MODULE=on`.

### Data Provisioning Issues

**Error: "Marine zones database not found" keeps appearing**

If the database fails to download or build, try manual provisioning:

```bash
# Remove any partial files
rm -rf data/

# Check internet connection
curl -I https://www.weather.gov/source/gis/Shapefiles/WSOM/mz18mr25.zip

# Run with verbose output
GO111MODULE=on go run test_nearby_zones.go
```

**Database file is corrupted**

Remove and re-provision:
```bash
rm -rf data/
./mariner-tui  # Will auto-provision on startup
```

**Slow download**

The shapefile is ~12 MB and downloads from NOAA servers. If the download is slow:
- Check your internet connection
- Try again later (NOAA servers may be busy)
- The download only happens once

## Acknowledgments

- NOAA for providing free marine weather data APIs
- Charm.sh for the excellent Bubble Tea framework
