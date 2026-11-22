package main

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type Change struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

func scanFolder(folder string) (map[string]string, error) {
	info, err := os.Stat(folder)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s must be a folder", folder)
	}

	files := make(map[string]string)

	err = filepath.WalkDir(folder, func(
		path string,
		entry fs.DirEntry,
		walkErr error,
	) error {
		if walkErr != nil {
			return walkErr
		}

		if !entry.Type().IsRegular() {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		relativePath, err := filepath.Rel(folder, path)
		if err != nil {
			return err
		}

		hash := sha256.Sum256(content)
		files[filepath.ToSlash(relativePath)] = fmt.Sprintf("%x", hash)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

func detectChanges(previous, current map[string]string) []Change {
	var changes []Change

	for path, currentHash := range current {
		previousHash, existed := previous[path]

		if !existed {
			changes = append(changes, Change{Type: "ADDED", Path: path})
		} else if previousHash != currentHash {
			changes = append(changes, Change{Type: "EDITED", Path: path})
		}
	}

	for path := range previous {
		if _, exists := current[path]; !exists {
			changes = append(changes, Change{Type: "DELETED", Path: path})
		}
	}

	return changes
}

func main() {
	const folder = "test_files"

	previous, err := scanFolder(folder)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to start monitoring:", err)
		os.Exit(1)
	}

	fmt.Printf("Monitoring %s with %d starting files.\n", folder, len(previous))
	fmt.Println("Checking every 2 seconds. Press Ctrl+C to stop.")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		current, err := scanFolder(folder)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Scan failed. Will retry:", err)
			continue
		}

		changes := detectChanges(previous, current)

		for _, change := range changes {
			fmt.Printf(
				"[%s] %s: %s\n",
				time.Now().Format("15:04:05"),
				change.Type,
				change.Path,
			)
		}

		previous = current
	}
}
