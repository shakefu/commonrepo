// Package gitutil provides helpers for dealing with git repositories. This
// should probably be an internal package but oh well.
package gitutil

import (
	"fmt"
	"os"
	stdpath "path"
	"path/filepath"
)

// FindLocalRepoPath returns the full path to the repository containing the
// current working directory. This is not really useful for anything except
// providing a shortcut to that path for tests, and maybe automation tools.
func FindLocalRepoPath() (path string, err error) {
	// Find our current directory
	if path, err = os.Getwd(); err != nil {
		return
	}
	// Find the parent which is our git root
	if path, err = detectGitPath(path); err != nil {
		return
	}
	// If the path isn't empty, we're good
	if path != "" {
		return
	}

	return "", fmt.Errorf("unable to find repo root")
}

// detectGitPath returns the repository root of the specified path, if there is
// one.
func detectGitPath(path string) (string, error) {
	// normalize the path
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	for {
		fi, err := os.Stat(stdpath.Join(path, ".git"))
		if err == nil {
			if !fi.IsDir() {
				return "", fmt.Errorf(".git exist but is not a directory")
			}
			return path, nil
		}
		if !os.IsNotExist(err) {
			// unknown error
			return "", err
		}

		if parent := filepath.Dir(path); parent == path {
			return "", fmt.Errorf(".git not found")
		} else {
			path = parent
		}
	}
}
