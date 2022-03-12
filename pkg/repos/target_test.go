package repos_test

import (
	"bytes"
	"testing"

	. "github.com/shakefu/commonrepo/pkg/repos"

	. "github.com/onsi/gomega"
	// . "github.com/shakefu/commonrepo/internal/testutil"
	"github.com/shakefu/goblin"
)

func TestTarget(t *testing.T) {
	// Initialize the Goblin test suite
	g := goblin.Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) }) // Gomega hook

	g.Describe("Target", func() {
		var repo *Repo
		var err error
		var targets map[string]Target
		var target Target
		var buf *bytes.Buffer

		g.BeforeEach(func() {
			if repo, err = GetLocalRepo(); err != nil {
				g.FailNow()
			}
			if targets, err = repo.GlobTargets("**/template.yml"); err != nil {
				g.FailNow()
			}
			var name string
			for name = range targets {
			}
			target = targets[name]
		})

		g.It("works", func() {
			// Sanity check
			Expect(target.Name).To(Equal("testdata/fixtures/templates/template.yml"))
			Expect(target.Vars).To(BeNil())
		})

		g.It("has a string representation that works for plain files", func() {
			target.Name = "filename"
			Expect(target.String()).To(Equal("<Repo.File:filename>"))
		})

		g.It("has a string representation that works for templates", func() {
			target.Name = "templatename"
			target.Vars = map[string]interface{}{"foo": "bar"}
			Expect(target.String()).To(Equal("<Repo.Template:templatename>"))
		})

		g.It("does a copy when no vars are set", func() {
			// Unrendered copy (no vars set)
			buf = new(bytes.Buffer)
			err = target.Write(buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(buf.String()).To(ContainSubstring("templated: {{"))
		})

		g.It("will error with missing vars", func() {
			buf = new(bytes.Buffer)
			target.Vars = map[string]interface{}{"foo": "bar"}
			err = target.Write(buf)
			Expect(err).To(HaveOccurred())
		})

		g.It("renders things happily", func() {
			buf = new(bytes.Buffer)
			target.Vars = map[string]interface{}{
				"project":   "commonrepo",
				"version":   "1.0.0",
				"templated": true,
			}
			err = target.Write(buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(buf.String()).To(ContainSubstring("templated: true"))
			Expect(buf.String()).To(ContainSubstring("project: commonrepo"))
		})
	})
}
