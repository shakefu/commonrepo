package files_test

import (
	"testing"

	. "github.com/shakefu/commonrepo/pkg/files"

	. "github.com/onsi/gomega"
	. "github.com/shakefu/commonrepo/internal/testutil"
	"github.com/shakefu/goblin"
)

func TestFiles(t *testing.T) {
	// Initialize the Goblin test suite
	g := goblin.Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) }) // Gomega hook

	g.Describe("List", func() {
		g.It("works", func() {
			repo := LocalRepo()
			files, err := List(repo.FS())
			Expect(err).ToNot(HaveOccurred())
			Expect(files).To(ContainElement(".commonrepo.yml"))
		})
	})
}
