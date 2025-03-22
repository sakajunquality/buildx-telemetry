package buildx

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/sakajunquality/buildx-telemetry/internal/logger"
	"go.uber.org/zap"
)

// BuildStep represents a single build step in the Docker build process
type BuildStep struct {
	Name      string
	Started   time.Time
	Completed time.Time
	Cached    bool
}

// Vertex represents a vertex in the build graph from buildx json output
type Vertex struct {
	Digest    string   `json:"digest"`
	Name      string   `json:"name"`
	Started   string   `json:"started,omitempty"`
	Completed string   `json:"completed,omitempty"`
	Inputs    []string `json:"inputs,omitempty"`
	Cached    bool     `json:"cached,omitempty"`
}

// LogEntry is the buildx build log output with --progress=rawjson option
type LogEntry struct {
	Vertexes []Vertex `json:"vertexes,omitempty"`
	Statuses []struct {
		ID        string `json:"id"`
		Vertex    string `json:"vertex"`
		Name      string `json:"name"`
		Current   int    `json:"current"`
		Timestamp string `json:"timestamp"`
		Started   string `json:"started,omitempty"`
		Completed string `json:"completed,omitempty"`
	} `json:"statuses,omitempty"`
}

// Parser handles parsing buildx logs
type Parser struct {
	reader io.Reader
	logger logger.Logger
}

// NewParser creates a new buildx log parser
func NewParser(reader io.Reader) *Parser {
	// Create a noop logger if none is provided
	noop, _ := logger.New(logger.DefaultConfig())
	return &Parser{
		reader: reader,
		logger: noop,
	}
}

// NewParserWithLogger creates a new buildx log parser with a logger
func NewParserWithLogger(reader io.Reader, log logger.Logger) *Parser {
	return &Parser{
		reader: reader,
		logger: log,
	}
}

// Parse reads the log stream and returns a slice of BuildStep
func (p *Parser) Parse() ([]BuildStep, error) {
	var steps []BuildStep
	scanner := bufio.NewScanner(p.reader)
	lineCount := 0
	vertexCount := 0

	p.logger.Debug("Starting to parse buildx log")

	for scanner.Scan() {
		lineCount++
		var entry LogEntry
		line := scanner.Text()

		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			p.logger.Debug("Failed to parse log line",
				zap.Int("line", lineCount),
				zap.Error(err))
			continue
		}

		for _, vertex := range entry.Vertexes {
			vertexCount++
			if vertex.Started != "" && vertex.Completed != "" {
				started, err := time.Parse(time.RFC3339Nano, vertex.Started)
				if err != nil {
					p.logger.Debug("Failed to parse start time",
						zap.String("vertex", vertex.Name),
						zap.Error(err))
					continue
				}
				completed, err := time.Parse(time.RFC3339Nano, vertex.Completed)
				if err != nil {
					p.logger.Debug("Failed to parse completion time",
						zap.String("vertex", vertex.Name),
						zap.Error(err))
					continue
				}

				duration := completed.Sub(started)
				step := BuildStep{
					Name:      vertex.Name,
					Started:   started,
					Completed: completed,
					Cached:    vertex.Cached,
				}
				steps = append(steps, step)

				p.logger.Debug("Parsed build step",
					zap.String("step", vertex.Name),
					zap.Duration("duration", duration),
					zap.Bool("cached", vertex.Cached))
			}
		}
	}

	p.logger.Info("Completed parsing build log",
		zap.Int("lines", lineCount),
		zap.Int("vertexes", vertexCount),
		zap.Int("steps", len(steps)))

	return steps, scanner.Err()
}

// PrintSteps prints the build steps in a human-readable format
func PrintSteps(steps []BuildStep) {
	for _, step := range steps {
		fmt.Printf("%s\nStarted: %s\nCompleted: %s\nCached: %v\n\n",
			step.Name,
			step.Started.Format(time.RFC3339Nano),
			step.Completed.Format(time.RFC3339Nano),
			step.Cached)
	}
}
