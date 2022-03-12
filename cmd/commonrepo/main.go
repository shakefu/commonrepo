package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/shakefu/commonrepo"
	"github.com/shakefu/commonrepo/pkg/gitutil"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/docopt/docopt-go"
	"github.com/kataras/golog"
)

// Build-time vars
var (
	Version   = "development"
	GoVersion string
	GitCommit string
	BuildTime string
	Platform  string
	BinName   = "commonrepo"
	Usage     string
)

func init() {
	golog.Default.TimeFormat = "2006-01-02 15:04:05.000"
	// Enable global Debug output if Environment forces it
	if os.Getenv("DEBUG") != "" {
		golog.SetLevel("debug")
	}

	// Get our command name if it wasn't specified at build time
	if BinName == "" {
		var err error // Hoist this scope, so we don't override BinName with a :=
		if BinName, err = os.Executable(); err != nil {
			golog.Fatal("Unable to retrieve executable name")
		}
		BinName = filepath.Base(BinName)
	}

	// Try to get a local branch name instead of "development" for the version
	if Version == "development" {
		ref, _ := gitutil.LocalBranch()
		if ref != "" {
			Version = ref.String()
		}
	}

	// Create our CLI help string
	usage := heredoc.Doc(`
        Monitor replication status of mysql databases.

        Usage:
            %[1]s -h|--help
            %[1]s --version
            %[1]s

        Options:
            -d, --debug                               show debug output
            -h, --help                                show this help
            --version                                 show the version
    `)

	// Inject binary name into help
	Usage = fmt.Sprintf(usage, BinName)
}

// Args gives easy access and checking for our CLI
type Args struct {
	Debug   bool
	Help    bool
	Version bool
}

// GetArgs returns the CLI args as a struct
func GetArgs(usage string, argv []string) (args *Args, err error) {
	// Parsing CLI
	parsed, err := docopt.ParseArgs(usage, argv, makeVersion())
	if err != nil {
		return
	}

	// Bind to our struct for easy args
	args = &Args{}
	if err = parsed.Bind(args); err != nil {
		return
	}

	// Set debug output if we had CLI args
	if args.Debug {
		golog.SetLevel("debug")
	}

	return
}

// Run actually executes the CLI once the args have been parsed and logs set and
// all the other things that need to happen.
func Run(args *Args) (err error) {
	golog.Info("We're running")
	err = DefaultRun()
	return
}

// DefaultRun does a bunch of default settings ... mostly for testing
// TODO: Add a dry-run flag
// TODO: Add sensible logging across the whole thing
// TODO: Debug why it just says "remote repository is empty"
func DefaultRun() (err error) {
	repoRoot, err := gitutil.FindLocalRepoPath()
	if err != nil {
		return
	}
	cr, err := commonrepo.New(repoRoot)
	if err != nil {
		return
	}
	err = cr.Init()
	if err != nil {
		return
	}
	composite := cr.Composite()
	err = composite.Write()
	return
}

// main is the CLI entrypoint
func main() {
	// Get the CLI args
	args, err := GetArgs(Usage, os.Args[1:])
	if err != nil {
		golog.Child(BinName).Fatal(err)
	}

	// Invoke the actual work
	if err = Run(args); err != nil {
		golog.Child(BinName).Fatal(err)
	}
}

// makeVersion returns the full version string for docopt to display.
func makeVersion() string {
	return fmt.Sprintf(heredoc.Doc(`
        %s:
         Version:    %s
         Go version: %s
         Git commit: %s
         Built:      %s
         OS/Arch:    %s`),
		BinName,
		Version,
		GoVersion,
		GitCommit,
		BuildTime,
		Platform,
	)
}
