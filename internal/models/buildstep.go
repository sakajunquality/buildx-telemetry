package models

import (
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
