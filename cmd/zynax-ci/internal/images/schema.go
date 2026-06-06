// SPDX-License-Identifier: Apache-2.0

// Package images provides types and functions for managing pinned container
// image digests in images/images.yaml.
package images

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ImageEntry represents one pinned container image in images/images.yaml.
type ImageEntry struct {
	Name      string   `yaml:"name"`
	Ref       string   `yaml:"ref"`
	Tag       string   `yaml:"tag"`
	Digest    string   `yaml:"digest"`
	Consumers []string `yaml:"consumers"`
}

// File is the parsed representation of images/images.yaml.
type File struct {
	Images []ImageEntry `yaml:"images"`
}

// Load reads and parses images/images.yaml from repoRoot.
func Load(repoRoot string) (File, error) {
	path := filepath.Join(repoRoot, "images", "images.yaml")
	data, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		return File{}, fmt.Errorf("images: load %s: %w", path, err)
	}
	var f File
	if err := yaml.Unmarshal(data, &f); err != nil {
		return File{}, fmt.Errorf("images: parse %s: %w", path, err)
	}
	return f, nil
}
