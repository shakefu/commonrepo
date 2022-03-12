// Package repos provides the Repo type
package repos

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gobwas/glob"
	"github.com/shakefu/commonrepo/pkg/common"
	"github.com/shakefu/commonrepo/pkg/config"
	"github.com/shakefu/commonrepo/pkg/files"
	"github.com/shakefu/commonrepo/pkg/gitutil"

	billy "github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
)

// New returns a new Repo instance from the URL and tarrge ref
func New(url string, refs ...string) (repo *Repo, err error) {
	repo = &Repo{
		URL: url,
		Ref: append(refs, "")[0],
	}
	if err = repo.Init(); err != nil {
		return nil, err
	}
	if err = repo.Clone(); err != nil {
		return nil, err
	}
	return
}

// Repo is an in-memory git repository.
type Repo struct {
	// Requested URL and git ref, these may not be the same as actual
	URL string
	Ref string
	// Actual URL, git ref, options used to clone, and low-level Repository
	url  string
	ref  plumbing.ReferenceName
	opts *git.CloneOptions
	repo *git.Repository
	// Filesystem and storage for the repository
	fs    billy.Filesystem
	store *memory.Storage
	files []string
	// Target files map
	targets map[string]Target
	// State flags
	inited bool
	cloned bool
}

// Init creates the memory filesystem and storage for the repository.
//
// This will make network requests to find the default branch, as well as read
// the filesystem to load the gitconfig.
func (repo *Repo) Init() (err error) {
	if repo.inited {
		return errors.New("repo already initialized")
	}

	// Create our storage
	repo.fs = memfs.New()
	repo.store = memory.NewStorage()

	// Load our system/global configs so we can apply the InsteadOf rules to our
	// url before we try to clone. This lets us inject credentials for private
	// repositories without having to make some custom auth scheme
	if repo.url, err = gitutil.ApplyInsteadOf(repo.URL); err != nil {
		return
	}

	// We want to either use the default branch (main/master) or figure out if
	// the ref we were given is a tag or a branch
	if repo.ref, err = gitutil.FindRef(repo.url, repo.Ref); err != nil {
		return
	}

	// Make our options for cloning
	repo.opts = &git.CloneOptions{
		URL:           repo.url,
		ReferenceName: repo.ref,
		SingleBranch:  true,
		Depth:         1,
	}

	// Populate Ref if it was empty and we used the default
	if repo.Ref == "" {
		ref := strings.Split(repo.ref.String(), "/")
		ref = ref[2:]
		repo.Ref = strings.Join(ref, "/")
	}

	// Set our init success flag
	repo.inited = true
	return
}

// Clone the given repository into this Repo and populate the list of files.
func (repo *Repo) Clone() (err error) {
	if repo.cloned {
		return errors.New("repo already cloned")
	}
	repo.repo, err = git.Clone(repo.store, repo.fs, repo.opts)
	if err != nil {
		return
	}
	repo.files, err = repo.list()

	// Initialize the renamed map to default
	repo.ResetTargets()

	repo.cloned = true
	return
}

// Check makes sure the Repo has been initialized and cloned and is ready.
func (repo *Repo) Check() (err error) {
	if !repo.inited {
		return errors.New("repo not initialized")
	}
	if !repo.cloned {
		return errors.New("repo not cloned")
	}
	if repo.targets == nil {
		repo.ResetTargets()
	}
	return
}

// list returns a list of all files in the repo
func (repo *Repo) list() (found []string, err error) {
	if !repo.inited {
		return nil, errors.New("repo not initialized")
	}

	found, err = files.List(repo.fs)
	return
}

// Glob returns a list of file names matching the given pattern.
func (repo *Repo) Glob(pattern string) (matches []string, err error) {
	if err = repo.Check(); err != nil {
		return
	}

	// Compile our matching glob... slow for small repos, but much faster for
	// very very large ones
	var g glob.Glob
	g, err = glob.Compile(pattern, filepath.Separator)
	if err != nil {
		return
	}
	// Iterate through the files and try to find our matches
	for _, file := range repo.files {
		if g.Match(file) {
			matches = append(matches, file)
		}
	}
	return
}

// GlobTargets returns a mapping of renamed filename to original filenames
// matching the given pattern.
func (repo *Repo) GlobTargets(pattern string) (matches map[string]Target, err error) {
	if err = repo.Check(); err != nil {
		return
	}

	// Compile our matching glob... slow for small repos, but much faster for
	// very very large ones
	var g glob.Glob
	g, err = glob.Compile(pattern, filepath.Separator)
	if err != nil {
		return
	}

	// Iterate through the files and try to find our matches
	matches = make(map[string]Target)
	for name, file := range repo.targets {
		if g.Match(name) {
			matches[name] = file
		}
	}
	return
}

// ResetTargets initializes the renamed map to the current list of files.
func (repo *Repo) ResetTargets() {
	repo.targets = make(map[string]Target, len(repo.files))
	for _, file := range repo.files {
		repo.targets[file] = Target{Name: file, repo: repo}
	}
}

// Targets returns the targets and initializes it if it hasn't been
func (repo *Repo) Targets() map[string]Target {
	if repo.targets == nil {
		repo.ResetTargets()
	}
	return repo.targets
}

// ApplyIncludes applies the given includes to the current targets.
//
// The returned value is the mapping of all targets.
func (repo *Repo) ApplyIncludes(includes []string) (found map[string]Target, err error) {
	if err = repo.Check(); err != nil {
		return
	}
	// Skip all this if there's no files to include
	if len(includes) == 0 || len(repo.targets) == 0 {
		found = make(map[string]Target, 0)
		repo.targets = found
		return
	}

	found = make(map[string]Target, len(repo.targets))
	var matched map[string]Target

	// Iterate over the renames applying them in order
	for _, include := range includes {
		matched, err = repo.GlobTargets(include)
		if err != nil {
			return
		}
		for k, v := range matched {
			found[k] = v
		}
	}
	repo.targets = found
	return
}

// ApplyExcludes applies the given excludes to the current targets.
//
// The returned value is the mapping of all targets.
func (repo *Repo) ApplyExcludes(excludes []string) (targets map[string]Target, err error) {
	if err = repo.Check(); err != nil {
		return
	}

	// Skip all this if there's no files or excludes
	if len(excludes) == 0 || len(repo.targets) == 0 {
		return repo.targets, nil
	}

	// Iterate over the renames applying them in order
	var matched map[string]Target
	for _, include := range excludes {
		matched, err = repo.GlobTargets(include)
		if err != nil {
			return
		}
		for k := range matched {
			delete(repo.targets, k)
		}
	}
	return repo.targets, nil
}

// ApplyTemplates creates the templates in our map of targets.
func (repo *Repo) ApplyTemplates(templates []string, templateVars map[string]interface{}) (err error) {
	if err = repo.Check(); err != nil {
		return
	}

	// Skip all this if there's no files or templates
	if len(templates) == 0 {
		return
	}

	// Iterate over the template globs adding them to our targets, regardless of
	// what was included/excluded previously
	var found []string
	for _, include := range templates {
		if found, err = repo.Glob(include); err != nil {
			return
		}
		for _, name := range found {
			repo.targets[name] = Target{Name: name, Vars: templateVars, repo: repo}
		}
	}
	return
}

// ApplyRenames applies the given renames.
//
// The returned value is the mapping of the renamed file to the original file.
// It can be mutated manually to change the internal state of the repo renames.
func (repo *Repo) ApplyRenames(renames []config.Rename) map[string]Target {
	if err := repo.Check(); err != nil {
		return repo.targets
	}

	// Skip all this if there's no files in the renamed map, for whatever reason
	if len(renames) == 0 || len(repo.targets) == 0 {
		return repo.targets
	}

	// Skip all this if there's no rename rules to parse
	if len(renames) == 0 {
		return repo.targets
	}

	// Grab the original keys of our targets map so we can modify it freely
	// without non-deterministic nonsense
	targets := SortTargetNames(repo.targets)

	// Iterate over the renames applying them in order
	for _, rename := range renames {
		// Iterate over the files applying our rename
		for _, name := range targets {
			if !rename.Check(name) {
				continue
			}
			rname := rename.Apply(name)
			if rname != name {
				// Save the rename
				repo.targets[rname] = repo.targets[name]
				// Remove the original entry
				delete(repo.targets, name)
			}
		}
	}

	// Return the updated rename map
	return repo.targets
}

// Stat returns fs.FileInfo from stat() on a file name
func (repo *Repo) Stat(name string) (os.FileInfo, error) {
	return repo.fs.Stat(name)
}

// FS returns the Repo's filesystem
func (repo *Repo) FS() billy.Filesystem {
	return repo.fs
}

// findConfig returns the shortest matching path in this Repo's files
func (repo *Repo) findConfig(search ...string) (path string, err error) {
	var pattern = common.ConfigFileGlob()
	if len(search) > 0 {
		pattern = search[0]
	}

	cfg, err := repo.Glob(pattern)
	if err != nil {
		return
	}
	if len(cfg) < 1 {
		err = errors.New("no config file found")
		return
	}

	sort.Slice(cfg, func(i, j int) bool {
		return len(cfg[i]) < len(cfg[j])
	})

	path = cfg[0]
	return
}

// readConfig finds and returns the yaml content of the config in this Repo if
// it exists.
func (repo *Repo) readConfig(search ...string) (yaml []byte, err error) {
	// Read the config
	var path string
	path, err = repo.findConfig(search...)
	if err != nil {
		return
	}

	yaml, err = repo.ReadFile(path)
	if err != nil {
		return
	}

	return
}

// LoadConfig returns the config in this Repo if it exists.
func (repo *Repo) LoadConfig(search ...string) (cfg *config.Config, err error) {
	yaml, err := repo.readConfig(search...)
	if err != nil {
		return
	}

	cfg, err = config.ParseConfig(yaml)
	return
}

// Open returns a file handle for the given file name.
func (repo *Repo) Open(name string) (io.Reader, error) {
	return repo.fs.Open(name)
}

// ReadFile reads the named file and returns the contents.
// A successful call returns err == nil, not err == EOF.
// Because ReadFile reads the whole file, it does not treat an EOF from Read
// as an error to be reported.
//
// This implementation is stolen from os.ReadFile
func (repo *Repo) ReadFile(name string) ([]byte, error) {
	f, err := repo.fs.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var size int
	if info, err := repo.fs.Stat(name); err == nil {
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

// GetLocalRepo returns a local Repo reference assuming we're executing this
// somewhere in the context of a git repository structure.
func GetLocalRepo() (rep *Repo, err error) {
	path, err := gitutil.FindLocalRepoPath()
	if err != nil {
		return
	}
	rep, err = New(path)
	return
}

// SortTargetNames returns the sorted keys of a target map
func SortTargetNames(targets map[string]Target) []string {
	names := make([]string, 0, len(targets))
	for name := range targets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
