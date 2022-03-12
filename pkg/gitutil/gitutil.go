// Package gitutil provides helpers for dealing with git repositories.
package gitutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/client"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/imdario/mergo"
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
	if path, err = DetectGitPath(path); err != nil {
		return
	}
	// If the path isn't empty, we're good
	if path != "" {
		return
	}

	return "", fmt.Errorf("unable to find repo root")
}

// DetectGitPath returns the repository root of the specified path, if there is
// one.
func DetectGitPath(path string) (string, error) {
	// normalize the path
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	for {
		fi, err := os.Stat(filepath.Join(path, ".git"))
		if err == nil {
			if !fi.IsDir() {
				return "", fmt.Errorf(".git exists but is not a directory")
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

// DefaultRef returns a ref pointing at the default HEAD for the repository,
// since not all repositories will use "main", this will determine it for us.
func DefaultRef(url string) (ref plumbing.ReferenceName, err error) {
	return FindRef(url, "")
}

// LocalBranch returns the current ref for this repository.
func LocalBranch() (ref plumbing.ReferenceName, err error) {
	repoRoot, err := FindLocalRepoPath()
	if err != nil {
		return
	}
	repository, err := git.PlainOpen(repoRoot)
	if err != nil {
		return
	}
	refs, err := repository.Branches()
	if err != nil {
		return
	}
	headRef, err := repository.Head()
	if err != nil {
		return
	}
	err = refs.ForEach(func(each *plumbing.Reference) error {
		if each.Hash() == headRef.Hash() {
			ref = each.Name()
			return nil
		}
		return nil
	})
	if err != nil {
		return
	}

	ref = plumbing.ReferenceName(strings.TrimPrefix(ref.String(), "refs/heads/"))
	return
}

// RemoteDefaultBranch returns the current ref for this repository.
func RemoteDefaultBranch() (ref plumbing.ReferenceName, err error) {
	repoRoot, err := FindLocalRepoPath()
	if err != nil {
		return
	}
	repository, err := git.PlainOpen(repoRoot)
	if err != nil {
		return
	}
	// TODO: Determine if we should support other remote names
	remote, err := repository.Remote("origin")
	if err != nil {
		return
	}
	remoteRefs, err := remote.List(&git.ListOptions{})
	if err != nil {
		return
	}
	for _, remoteRef := range remoteRefs {
		if remoteRef.Name() == "HEAD" {
			ref = remoteRef.Target()
		}
	}

	ref = plumbing.ReferenceName(strings.TrimPrefix(ref.String(), "refs/heads/"))
	return
}

// FindRef returns a ref from the given refname, or falls back to the default
// branch for the repository.
func FindRef(url string, refname string) (ref plumbing.ReferenceName, err error) {
	refs, err := GetRefs(url)
	if err != nil {
		return
	}

	// Handle the default case without processing all the refs
	if refname == "" {
		// Pull out just the default HEAD ref
		ref = refs["HEAD"].Target()
		return
	}

	names := []plumbing.ReferenceName{
		plumbing.ReferenceName(refname),
		plumbing.NewTagReferenceName(refname),
		plumbing.NewBranchReferenceName(refname),
	}
	for _, ref := range names {
		if _, ok := refs[ref]; ok {
			return ref, nil
		}
	}
	ref = refs["HEAD"].Target()
	return
}

// GetRefs returns a map of all the available refs
//
// This is borrowed from:
// https://github.com/go-git/go-git/issues/249#issuecomment-772354474
func GetRefs(url string) (refs memory.ReferenceStorage, err error) {
	// Get the endpoint config and determine transport type
	end, err := transport.NewEndpoint(url)
	if err != nil {
		return
	}

	// Get a client instance for the given transport type
	cli, err := client.NewClient(end)
	if err != nil {
		return
	}

	// Invoke the endpoint agains the client we just got to get a session
	sess, err := cli.NewUploadPackSession(end, nil)
	if err != nil {
		return
	}

	// Get all the references
	info, err := sess.AdvertisedReferences()
	if err != nil {
		return
	}

	refs, err = info.AllReferences()
	return
}

// ApplyInsteadOf returns the remote URL with the instead of rules applied.
func ApplyInsteadOf(remoteURL string) (url string, err error) {
	url = remoteURL
	gitcfg, err := LoadGitConfig()
	if err != nil {
		return
	}
	// See if the url matches any of our rules
	match := FindInsteadOfMatch(url, gitcfg.URLs)
	if match != nil {
		// And if we found 'em apply em
		url = match.ApplyInsteadOf(url)
	}
	return
}

// FindInsteadOfMatch returns the longest matching URL rule
//
// This is borrowed directly from go-git:
// https://github.com/go-git/go-git/blob/master/config/url.go#L59
func FindInsteadOfMatch(remoteURL string, urls map[string]*config.URL) *config.URL {
	var longestMatch *config.URL
	for _, u := range urls {
		if !strings.HasPrefix(remoteURL, u.InsteadOf) {
			continue
		}

		// according to spec if there is more than one match, take the logest
		if longestMatch == nil || len(longestMatch.InsteadOf) < len(u.InsteadOf) {
			longestMatch = u
		}
	}

	return longestMatch
}

// LoadGitConfig returns the system and global git configs merged together.
//
// This is borrowed directly from go-git:
// https://github.com/go-git/go-git/blob/master/repository.go#L489
func LoadGitConfig() (conf *config.Config, err error) {
	system, err := config.LoadConfig(config.SystemScope)
	if err != nil {
		return nil, err
	}

	conf, err = config.LoadConfig(config.GlobalScope)
	if err != nil {
		return nil, err
	}

	_ = mergo.Merge(conf, system)
	return conf, nil
}
