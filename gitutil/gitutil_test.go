package gitutil

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	goblin "github.com/shakefu/goblin"
)

func TestGitUtil(t *testing.T) {
	g := goblin.Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) }) // Gomega hook
	g.Describe("FindLocalRepoPath", func() {
		g.It("should work", func() {
			repo, err := FindLocalRepoPath()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(repo).ShouldNot(Equal(""))
			Expect(filepath.Base(repo)).To(Equal("commonrepo"))
		})
	})
}
