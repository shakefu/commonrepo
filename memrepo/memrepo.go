// Package memrepo provides an in-memory repository implementation for fast
// cloning remotes.
package memrepo

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kataras/golog"

	billy "github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/util"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/spf13/cast"
)

var log *golog.Logger

func init() {
	log = golog.Child("memgit")
}

// The golang guides claim you shouldn't alias basic types like this because
// aliases are only intended for refactoring steps, and not for day to day use.
// But I say that's dumb. This creates more readable code and lets you do Rename
// operations on composite types that otherwise causes IDEs to balk. Fight me.

// Yaml interface helper type for functions which deal with ambiguous YAML configuration.
type Yaml = interface{}

// Dict interface helper type for the common YAML configuration.
type Dict = map[string]interface{}

// The sensible configuration for cloning repositories into memory
var gitConfigDefaults = Dict{
	// TBD: This could be "master" instead, but meh
	"referencename": "refs/heads/main",
	"singlebranch":  true,
	"depth":         1,
}

// Actual git data pulled from clone
type memGit struct {
	fs         billy.Filesystem
	store      *memory.Storage
	gitOptions *git.CloneOptions
}

// Repo interface to a cloned repository instance.
type Repo interface {
	Glob(string) ([]string, error)
	ReadFile(string) ([]byte, error)
	Stat(string) (os.FileInfo, error)
}

// NewRepo returns a pointer to a memGit instance that has cloned the repository
// specified, or nil if there is an issue.
//
// This function takes any number of `map[string]interface{}` configs which are
// composed into one map before being translated into the *git.CloneOptions
// used to clone the repo.
//
// Key names are case insensitive.
//
// If the git configuration is bad or another issue arises, nil is returned.
func NewRepo(args ...Yaml) (Repo, error) {
	var this *memGit
	// Parse all the garbage we got into some config for the clone
	opts, err := makeGitCloneOptions(args...)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// Do some nice validation once we've got this far
	err = opts.Validate()
	if err != nil {
		log.Error("Invalid git config: ", err)
		return nil, err
	}

	// Create our new storage in memory for the repo
	this = &memGit{
		memfs.New(),
		memory.NewStorage(),
		opts,
	}

	// Clone the repo into ourselves
	err = this.clone(opts)
	if err != nil {
		log.Error("Unable to clone repository: ", err)
		return this, err
	}

	return this, err
}

// Clone the given repository into this.
func (this *memGit) clone(args *git.CloneOptions) error {
	golog.Debug("Cloning...")
	_, err := git.Clone(this.store, this.fs, args)
	return err
}

// Glob returns a list of file names matching the given pattern.
func (this *memGit) Glob(pattern string) (matches []string, err error) {
	return util.Glob(this.fs, pattern)
}

// Stat returns fs.FileInfo from stat() on a file name
func (this *memGit) Stat(name string) (os.FileInfo, error) {
	return this.fs.Stat(name)
}

// ReadFile reads the named file and returns the contents.
// A successful call returns err == nil, not err == EOF.
// Because ReadFile reads the whole file, it does not treat an EOF from Read
// as an error to be reported.
//
// This implementation is stolen from os.ReadFile
func (this *memGit) ReadFile(name string) ([]byte, error) {
	f, err := this.fs.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var size int
	if info, err := this.fs.Stat(name); err == nil {
		size64 := info.Size()
		if int64(int(size64)) == size64 {
			size = int(size64)
		}
	}
	size++ // one byte for final read at EOF

	// If a file claims a small size, read at least 512 bytes.
	// In particular, files in Linux's /proc claim size 0 but
	// then do not work right if read in small pieces,
	// so an initial read of 1 byte would not work correctly.
	if size < 512 {
		size = 512
	}

	data := make([]byte, 0, size)
	for {
		if len(data) >= cap(data) {
			d := append(data[:cap(data)], 0)
			data = d[:len(data)]
		}
		n, err := f.Read(data[len(data):cap(data)])
		data = data[:len(data)+n]
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return data, err
		}
	}
}

// makeGitCloneOptions returns *git.CloneOptions from args and defaults.
func makeGitCloneOptions(args ...Yaml) (opts *git.CloneOptions, err error) {
	// Process the Yaml into a consistent Dict, with defaults
	config, err := makeGitConfig(args...)
	if err != nil {
		return nil, fmt.Errorf("could not make git config: %w", err)
	}

	// Encode the config as JSON as a precursor to unmarshling into our
	// git.CloneOptions
	body, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("could not encode git config: %w", err)
	}

	// Unpack the JSON to map fields easily
	err = json.Unmarshal(body, &opts)
	if err != nil {
		return nil, fmt.Errorf("could not make git.CloneOptions: %w", err)
	}

	return
}

// makeGitConfig returns a dict containing all lower case keys for a
// configuration corresponding to git.CloneOptions.
//
// This will incorporate the defaults defned in this package, as well as those
// passed in via the YAML configuration.
func makeGitConfig(args ...Yaml) (config Dict, err error) {
	// Gather our defaults and args into this
	config = Dict{}

	// Systemwide defaults
	copyDictLower(config, gitConfigDefaults)

	// Loop over all the dict-ish things we got
	for _, conf := range args {
		if conf == nil {
			continue
		}
		conf, err := gitURLorConfig(conf)
		if err != nil {
			return config, fmt.Errorf("invalid git config: %w", err)
		}
		copyDictLower(config, conf)
	}

	return
}

// gitURLorConfig returns a Dict from arg regardless if it's a Dict or a string.
//
// If a string is passed, it will be assigned to the "url" key in the returned
// Dict.
func gitURLorConfig(arg Yaml) (Dict, error) {
	// Try to see if our arg is a string URI
	url, err := cast.ToStringE(arg)
	if err == nil {
		return Dict{"url": url}, nil
	}

	return cast.ToStringMapE(arg)
}

// copyDictLower does a simple copy by key merge of source into dest while
// lowercasing all the key names.
//
// This is used to merge the git configuration YAML and defaults together into
// the final configuration.
func copyDictLower(dest Dict, source Dict) {
	// Yey it's a map, merge it
	for k, v := range source {
		k = strings.ToLower(k)
		if _, ok := dest[k]; ok && v == nil {
			// The value is overwriting the default with nil, e.g. removing
			// it, so let's help out by deleting it instead
			delete(dest, k)
			continue
		}
		dest[k] = v
	}
}
