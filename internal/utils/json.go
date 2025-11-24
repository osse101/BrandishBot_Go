package utils

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadJSON reads a JSON file and unmarshals it into the target interface.
func LoadJSON(path string, target interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal JSON from %s: %w", path, err)
	}
	return nil
}

// SaveJSON marshals the data and writes it to a JSON file.
func SaveJSON(path string, data interface{}) error {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	if err := os.WriteFile(path, bytes, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}
	return nil
}
