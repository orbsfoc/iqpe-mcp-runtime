package io

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func WriteYAML(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
