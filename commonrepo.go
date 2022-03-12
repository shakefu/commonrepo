// Package commonrepo contains the main entrypoint for working with the
// framework
package commonrepo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"go.uber.org/multierr"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/shakefu/commonrepo/pkg/common"
	"github.com/shakefu/commonrepo/pkg/config"
	"github.com/shakefu/commonrepo/pkg/gitutil"
	"github.com/shakefu/commonrepo/pkg/repos"
)

// CommonRepo provides the top level interface for operations
type CommonRepo struct {
	// Options which can be changed at runtime
	MaxUpstreamDepth int // How deep we will keep cloning upstreams (default: 5)
	// Internal
	repo      *repos.Repo    // The repo cloned as a source
	config    *config.Config // The configuration loaded from the repo
	upstreams []*CommonRepo  // Upstream CommonRepo tree
	flattened []*CommonRepo  // Upstreams flattened into ordered list with self
	from      string         // The original path of the loaded configuration
}

// New returns a new CommonRepo loading the default configuration glob.
//
// This is how this should normally be used.
func New(url string, ref ...string) (cr *CommonRepo, err error) {
	cr, err = NewFrom(common.ConfigFileGlob(), url, ref...)
	return
}

// NewFrom returns a new CommonRepo loading the given configuration search glob.
//
// This is mostly useful for testing, since we don't have to use our root
// configuration.
func NewFrom(from string, url string, ref ...string) (cr *CommonRepo, err error) {
	repo, err := repos.New(url, ref...)
	if err != nil {
		return
	}

	cr, err = NewFromRepo(from, repo)
	if err != nil {
		return
	}
	return
}

// NewFromRepo returns a new CommonRepo using the given Repo and from search
// glob.
//
// This might be useless but I added it anyway. I'll delete it later if I don't
// need it.
func NewFromRepo(from string, repo *repos.Repo) (cr *CommonRepo, err error) {
	config, err := repo.LoadConfig(from)
	if err != nil {
		return
	}

	cr = &CommonRepo{
		repo:   repo,
		config: config,
		from:   from,
	}
	cr.setDefaultOptions()
	return
}

// NewFromRename returns a new CommonRepo using the given Repo and renames which
// might modify the base path for the .commonrepo.yml
func NewFromRename(repo *repos.Repo, renames []config.Rename) (cr *CommonRepo, err error) {
	// Apply the renames
	repo.ApplyRenames(renames)
	matches, err := repo.GlobTargets(common.ConfigFileGlob())
	if err != nil {
		return
	}

	// Clear the renames so we can preserve order of the config files' renames
	repo.ResetTargets()

	// No matches for a commonrepo config is... a bug? Probably? But we provide
	// a sane default anyway
	// TODO: Allow non-CommonRepo upstreams or just fail? Maybe an option?
	if len(matches) == 0 {
		// Make an empty config instance so it initializes the defaults
		var empty *config.Config
		empty, err = config.ParseConfig([]byte{})
		if err != nil {
			return
		}
		// Use it to make our CommonRepo
		cr = &CommonRepo{
			repo:   repo,
			config: empty,
		}
		cr.setDefaultOptions()
		return
	}

	// Get the shortest match from the renamed files, since at least that's
	// probably deterministic
	keys := make([]string, 0, len(matches))
	for k := range matches {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i int, j int) bool {
		return len(keys[i]) < len(keys[j])
	})
	target := keys[0]

	// Original file name
	found := matches[target]
	// Load the CommonRepo from the config that we found
	cr, err = NewFromRepo(found.Name, repo)
	return
}

func (cr *CommonRepo) Init() (err error) {
	// Load all the upstreams
	// TODO: Skip this if already loaded?
	if err = cr.LoadUpstreams(cr.MaxUpstreamDepth); err != nil {
		return
	}
	// Get the flattened list of upstreams
	// TODO: Skip this if already populated?
	cr.flattened = cr.FlattenUpstreams()

	// Composite all our template vars into a single map
	templateVars := make(map[string]interface{}, 16)
	for _, each := range cr.flattened {
		for k, v := range each.config.TemplateVars {
			templateVars[k] = v
		}
	}

	// Apply the configs to each repo
	for _, each := range cr.flattened {
		if _, err = each.repo.ApplyIncludes(each.config.Include); err != nil {
			return
		}

		if err = each.repo.ApplyTemplates(each.config.Template, templateVars); err != nil {
			return
		}

		if _, err = each.repo.ApplyExcludes(each.config.Exclude); err != nil {
			return
		}

		each.repo.ApplyRenames(each.config.Rename)
	}
	return
}

// Upstreams loads and flattens all the upstream repos into their inheritance
// order.
func (cr *CommonRepo) Upstreams() (upstreams []*CommonRepo, err error) {
	// Ensure the upstreams are loaded
	if len(cr.upstreams) == 0 {
		if err = cr.LoadUpstreams(cr.MaxUpstreamDepth); err != nil {
			return
		}
	}
	// Ensure we've cached the flattened structure for easy reference
	if len(cr.flattened) == 0 {
		cr.flattened = cr.FlattenUpstreams()
	}
	// Return the list of flattened upstreams
	upstreams = cr.flattened
	return
}

// LoadUpstreams recursively clones all the upstream repositories
func (cr *CommonRepo) LoadUpstreams(depth int) (errs error) {
	var cloning sync.WaitGroup
	var repo *repos.Repo
	var err error

	// Terminating condition, we delved too greedily and too deep and awoke the
	// flame in the darkness
	if depth < 1 {
		return errors.New("maximum recursion depth reached")
	}

	// Terminating condition, we don't have any Upstream repositories to clone
	if len(cr.config.Upstream) == 0 {
		return
	}

	// Preallocate the list of upstreams
	cr.upstreams = make([]*CommonRepo, len(cr.config.Upstream))

	// Iterate through the upstreams in the config
	for i, upstream := range cr.config.Upstream {
		cloning.Add(1)

		// Schedule asynchronous cloning of the repository
		go func(upstream config.Upstream, i int) {
			defer cloning.Done()

			// Create a new Repo, initialize and clone it
			if repo, err = repos.New(upstream.URL, upstream.Ref); err != nil {
				errs = multierr.Append(errs, err)
				return
			}

			// Try to find a commonrepo config file, while applying the renames
			// we have defined in the parent's config for this upstream, if any
			if cr.upstreams[i], err = NewFromRename(repo, upstream.Rename); err != nil {
				errs = multierr.Append(errs, err)
				return
			}

			// Yeet this off into the scheduler
			cloning.Add(1)
			defer func(cr *CommonRepo) {
				defer cloning.Done()

				// Descend another layer into cloning
				err := cr.LoadUpstreams(depth - 1)
				if err != nil {
					errs = multierr.Append(errs, err)
					return
				}
			}(cr.upstreams[i])
		}(upstream, i)
	}
	cloning.Wait()
	// Ideally by the time we get here, the recursive cloning is done, yay
	return
}

// FlattenUpstreams returns a slice of all the upstreams flattened into the
// inherited order.
//
// This will mutate the config of the CommonRepo and its upstreams in order to
// apply the downstream include/exclude/rename rules.
func (cr *CommonRepo) FlattenUpstreams() (upstreams []*CommonRepo) {
	// If we don't have anything at all, just return the empty list
	if cr.upstreams == nil {
		return []*CommonRepo{cr}
	}

	// Preallocate the list of upstreams to the minimum length that it can be
	upstreams = make([]*CommonRepo, 0, len(cr.upstreams)+1)

	// Iterate through the upstreams that we inherit
	for i, upstream := range cr.upstreams {
		// Mutate the upstream's config to apply the downstream rules
		upstream.AppendConfig(&cr.config.Upstream[i])
		// Append the upstreams to our upstreams
		upstreams = append(upstreams, upstream.FlattenUpstreams()...)
	}

	// And finally append ourselves
	upstreams = append(upstreams, cr)
	return
}

// AppendConfig appends the given config.Upstream to this
func (cr *CommonRepo) AppendConfig(parent *config.Upstream) {
	cr.config.Include = append(cr.config.Include, parent.Include...)
	cr.config.Exclude = append(cr.config.Exclude, parent.Exclude...)
	cr.config.Rename = append(cr.config.Rename, parent.Rename...)
}

// String satisifes the stringer interface and returns repo/from@ref
func (cr *CommonRepo) String() string {
	return fmt.Sprintf("%s/%s@%s", cr.repo.URL, cr.from, cr.repo.Ref)
}

// Composite brings together all the upstreams into a single map of target file
// names to their contents
func (cr *CommonRepo) Composite() Composited {
	composited := make(Composited)

	// Iterate through all of our commonrepos in order
	for _, each := range cr.flattened {
		// Getting all the targets we have
		targets := each.repo.Targets()
		// fmt.Println(targets)
		// And mapping them to their paths
		for name := range targets {
			composited[name] = targets[name]
		}
	}

	return composited
}

func (cr *CommonRepo) setDefaultOptions() {
	cr.MaxUpstreamDepth = 5
}

// Composited exists as a type just so we can mount the Write* methods on it
type Composited map[string]repos.Target

// Write writes the composited targets to the repository root
func (composite Composited) Write() (err error) {
	var base string
	if base, err = gitutil.FindLocalRepoPath(); err != nil {
		return
	}
	// Grab a filesystem off of the repo root
	fs := osfs.New(base)

	// Write the targets to the filesystem
	if err = composite.WriteFS(fs, "/"); err != nil {
		return
	}

	return
}

// WriteFS writes the composite to the given filesystem
func (composite Composited) WriteFS(fs billy.Filesystem, basePaths ...string) (errs error) {
	var base string
	var err error
	if basePaths == nil || len(basePaths) < 1 {
		// Default to the local repository root.
		// TODO: This should not default to the repo root - instead we should
		// give it a filesystem that is defaulted to the repo root so you can't
		// write outside it
		base, err = gitutil.FindLocalRepoPath()
		if err != nil {
			return multierr.Append(errs, err)
		}
	} else {
		base = basePaths[0]
	}

	// We're going to try to do this asynchronously, for no other reason than
	// it's fun and ... maaaaaaybe it'll be slightly marginally faster for large
	// copies.
	var copying sync.WaitGroup
	var mkdir sync.Mutex

	// Iterate over all our targets and write them to the given filesystem
	for name, target := range composite {
		copying.Add(1)
		go func(name string, target repos.Target) {
			defer copying.Done()

			// Get the fullName which we will write to eventually
			fullName := filepath.Join(base, name)
			fullName = filepath.Clean(fullName)

			// TODO: Determine if we want to wall off writing outside the repo root
			// There's definitely a use case, e.g. for composing dotfiles into the
			// home directory, but there's also a lot of risk.

			// Make sure the write target path exists... we have to lock this
			// because multiple files could be trying to create the parent paths
			// simultaneously which the filesystem doesn't like at all
			mkdir.Lock()
			if err = fs.MkdirAll(filepath.Dir(fullName), 0755); err != nil {
				mkdir.Unlock()
				errs = multierr.Append(errs, err)
				return
			}
			mkdir.Unlock()

			// Get the info so we can create the file with the right permissions
			var info os.FileInfo
			if info, err = target.Stat(); err != nil {
				errs = multierr.Append(errs, err)
				return
			}
			var mode = info.Mode()

			// TODO: This won't work if we decide we want to do a YAML merge
			// Probably need to wrap the Mkdir/Stat/Open stuff to check for file
			// existence, etc.
			// TODO: Add flag to prevent overwriting files (optionally)
			// Copy the file data
			var handle billy.File
			handle, err = fs.OpenFile(fullName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
			if err != nil {
				errs = multierr.Append(errs, err)
				return
			}
			defer handle.Close()

			err = target.Write(handle)
			if err != nil {
				errs = multierr.Append(errs, err)
				return
			}
		}(name, target)
	}

	copying.Wait()

	return
}
