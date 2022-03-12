// Package runner contains all the stuff used to run scripts in memory.
package runner

import (
	"fmt"
	"io"
	"os"

	"github.com/shakefu/commonrepo/pkg/repos"
)

// These exist so we can patch them through tests
var runStdout io.Writer
var runStderr io.Writer

func init() {
	runStdout = os.Stdout
	runStderr = os.Stderr
}

// Run will invoke the script file found at path in the given repo.
func Run(path string, repo *repos.Repo) (err error) {
	data, err := repo.ReadFile(path)
	if err != nil {
		return
	}
	content := string(data)
	script := NewScript(path, content)
	script.Stdout = runStdout
	script.Stderr = runStderr
	exitcode, err := script.Run()
	if err != nil {
		return
	}
	if exitcode != 0 {
		return fmt.Errorf("script '%s' exited with code %d", path, exitcode)
	}
	return
}

// RunAll invokes all script files found matching glob in the given repo.
func RunAll(glob string, repo *repos.Repo) (err error) {
	paths, err := repo.Glob(glob)
	if err != nil {
		return
	}
	for _, path := range paths {
		err = Run(path, repo)
		if err != nil {
			return
		}
	}
	return
}
