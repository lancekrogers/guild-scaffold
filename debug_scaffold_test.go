package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"os/exec"
)

func main() {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "debug_scaffold")
	if err != nil {
		fmt.Printf("Failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)

	// Copy test files
	outputDir := filepath.Join(tempDir, "output")

	// Build scaffold binary
	binaryPath := filepath.Join(tempDir, "scaffold")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/scaffold")
	cmd.Dir = "."
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to build scaffold: %v\n", err)
		os.Exit(1)
	}

	// Copy the minimal.yaml fixture and templates
	fixtureSource := "tests/integration/fixtures/minimal.yaml"
	templateSource := "tests/integration/fixtures/templates/minimal"

	fixtureDest := filepath.Join(tempDir, "minimal.yaml")
	templateDest := filepath.Join(tempDir, "templates", "minimal")

	// Copy fixture
	if err := copyFile(fixtureSource, fixtureDest); err != nil {
		fmt.Printf("Failed to copy fixture: %v\n", err)
		os.Exit(1)
	}

	// Copy templates
	if err := copyDir(templateSource, templateDest); err != nil {
		fmt.Printf("Failed to copy templates: %v\n", err)
		os.Exit(1)
	}

	// Run scaffold command
	args := []string{
		"init", "my-project",
		"--template", fixtureDest,
		"--output", outputDir,
		"--var", "project_name=MyTestProject",
		"--var", "module_name=github.com/test/myproject",
	}

	fmt.Printf("Running: %s %v\n", binaryPath, args)
	cmd = exec.Command(binaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Scaffold command failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Scaffold completed successfully!\n")

	// List created files
	err = filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		fmt.Printf("Created: %s\n", path)
		return nil
	})
	if err != nil {
		fmt.Printf("Failed to list files: %v\n", err)
	}
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0644)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}