// Package commonrepo provides top level common functions that can be used by
// subpackages when there is no import cycle.
package commonrepo

import (
	"github.com/shakefu/commonrepo/gitutil"
	"github.com/shakefu/commonrepo/memrepo"
)

// GetLocalRepo returns a local Repo reference assuming we're executing this
// somewhere in the context of a git repository structure.
//
// This isn't really useful for anything except tests.
func GetLocalRepo() (repo memrepo.Repo, err error) {
	path, err := gitutil.FindLocalRepoPath()
	if err != nil {
		return
	}
	// repo, err = memrepo.NewRepo(fmt.Sprintf("file://%s", path))
	repo, err = memrepo.NewRepo(path)
	return
}
