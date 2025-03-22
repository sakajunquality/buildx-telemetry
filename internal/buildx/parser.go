package buildx

import (
	"bufio"
	"encoding/json"
	"io"
	"time"
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
}

// NewParser creates a new buildx log parser
func NewParser(reader io.Reader) *Parser {
	return &Parser{
		reader: reader,
	}
}

// Parse reads the log stream and returns a slice of BuildStep
func (p *Parser) Parse() ([]BuildStep, error) {
	var steps []BuildStep
	scanner := bufio.NewScanner(p.reader)
	for scanner.Scan() {
		var entry LogEntry
		if err := json.Unmarshal([]byte(scanner.Text()), &entry); err != nil {
			continue
		}

		for _, vertex := range entry.Vertexes {
			if vertex.Started != "" && vertex.Completed != "" {
				started, err := time.Parse(time.RFC3339Nano, vertex.Started)
				if err != nil {
					continue
				}
				completed, err := time.Parse(time.RFC3339Nano, vertex.Completed)
				if err != nil {
					continue
				}

				step := BuildStep{
					Name:      vertex.Name,
					Started:   started,
					Completed: completed,
					Cached:    vertex.Cached,
				}
				steps = append(steps, step)
			}
		}
	}
	return steps, scanner.Err()
}

// PrintSteps prints the build steps in a human-readable format
func PrintSteps(steps []BuildStep) {
	for _, step := range steps {
		println(step.Name)
		println("Started: " + step.Started.Format(time.RFC3339Nano))
		println("Completed: " + step.Completed.Format(time.RFC3339Nano))
		println("Cached: ", step.Cached)
		println("")
	}
}
