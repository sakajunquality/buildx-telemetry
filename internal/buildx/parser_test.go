package buildx

import (
	"strings"
	"testing"
	"time"
)

func TestParser_ParseEmpty(t *testing.T) {
	reader := strings.NewReader("")
	parser := NewParser(reader)
	steps, err := parser.Parse()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(steps) != 0 {
		t.Errorf("Expected empty steps, got %d steps", len(steps))
	}
}

func TestParser_ParseValidJson(t *testing.T) {
	jsonData := `
	{"vertexes":[{"name":"step1", "digest":"sha256:abc", "started":"2023-01-01T00:00:00Z", "completed":"2023-01-01T00:00:10Z", "cached":false}]}
	{"vertexes":[{"name":"step2", "digest":"sha256:def", "started":"2023-01-01T00:00:10Z", "completed":"2023-01-01T00:00:20Z", "cached":true}]}
	`

	reader := strings.NewReader(jsonData)
	parser := NewParser(reader)
	steps, err := parser.Parse()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(steps) != 2 {
		t.Errorf("Expected 2 steps, got %d steps", len(steps))
		return
	}

	if steps[0].Name != "step1" {
		t.Errorf("Expected step name 'step1', got '%s'", steps[0].Name)
	}

	if steps[0].Cached {
		t.Errorf("Expected step1 cached to be false")
	}

	if steps[1].Name != "step2" {
		t.Errorf("Expected step name 'step2', got '%s'", steps[1].Name)
	}

	if !steps[1].Cached {
		t.Errorf("Expected step2 cached to be true")
	}

	// Check timestamp parsing
	expectedStart, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
	if !steps[0].Started.Equal(expectedStart) {
		t.Errorf("Expected start time %v, got %v", expectedStart, steps[0].Started)
	}
}
