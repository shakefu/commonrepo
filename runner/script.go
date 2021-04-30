package runner

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/jinzhu/copier"
)

// Script represents an in-memory script that can be run.
type Script struct {
	name      string
	Content   string
	Env       []string
	Shell     string
	ShellArgs []string
	Stdout    io.Writer
	Stderr    io.Writer
}

// NewScript returns a pointer to a newly created script instance.
func NewScript(name string, content string) *Script {
	this := &Script{
		name:    name,
		Content: content,
		Env:     os.Environ(),
		// TODO: Embed this?
		Shell:     "/bin/dash",
		ShellArgs: []string{"-s", "-"},
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
	}
	this.Env = append(this.Env, fmt.Sprintf("NAME=%s", this.name))
	return this
}

// String name of the Script.
func (this *Script) String() string {
	return fmt.Sprintf("Script{%s}", this.name)
}

// AddEnv is a convenience function for adding a new environment variable to the
// Script.
func (this *Script) AddEnv(name string, value string) {
	this.Env = append(this.Env, fmt.Sprintf("%s=%s", name, value))
}

// Run executes the Script in a subshell.
//
// Multiple runs of the same Script may or may not work.
func (this *Script) Run(args ...string) (exitcode int, err error) {
	exitcode = -1

	// Set the script arguments
	var argv []string
	copier.Copy(&argv, this.ShellArgs)
	argv = append(argv, args...)

	// And create the Command to run it in
	cmd := exec.Command(this.Shell, argv...)
	cmd.Env = this.Env
	cmd.Stdout = this.Stdout
	cmd.Stderr = this.Stderr

	// Grab the pipe to the subprocess stdin
	var stdin io.WriteCloser
	if stdin, err = cmd.StdinPipe(); err != nil {
		return
	}

	// Start the subprocess
	if err = cmd.Start(); err != nil {
		return
	}

	// Write the script to stdin and close the pipe when finished
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, this.Content)
	}()

	// Get the error code from the subprocess if it exists
	if exitError, ok := cmd.Wait().(*exec.ExitError); ok {
		return exitError.ExitCode(), exitError
	}

	return 0, nil
}
