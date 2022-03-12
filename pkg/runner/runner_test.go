package runner

import (
	"bytes"
	"os"
	"testing"

	. "github.com/onsi/gomega"
	. "github.com/shakefu/commonrepo/internal/testutil"
	"github.com/shakefu/goblin"
)

func TestRunner(t *testing.T) {
	g := goblin.Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) }) // Gomega hook

	g.Describe("runner", func() {
		g.Describe("Run", func() {
			g.It("works to run an individual script", func() {
				runStdout = &bytes.Buffer{}
				repo := LocalRepo()
				err := Run("test/helloworld.sh", repo)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(runStdout.(*bytes.Buffer).String()).To(Equal("Hello World\n"))
			})

			g.After(func() {
				runStdout = os.Stdout
			})
		})

		g.Describe("RunAll", func() {
			g.It("runs all scripts it finds", func() {
				runStdout = &bytes.Buffer{}
				repo := LocalRepo()
				err := RunAll("test/hello*", repo)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(runStdout.(*bytes.Buffer).String()).To(Equal("Hello World\n"))
			})

			g.After(func() {
				runStdout = os.Stdout
			})
		})
	})
}
