package runner

import (
	"bytes"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/shakefu/goblin"
)

func TestScript(t *testing.T) {
	g := goblin.Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) }) // Gomega hook

	g.Describe("script", func() {
		g.Describe("NewScript", func() {
			var script *Script
			g.It("should work", func() {
				script = NewScript("test/NewScript", "echo Hello World")
				script.Stdout = &bytes.Buffer{}
				script.Run()
				Expect(script.Stdout.(*bytes.Buffer).String()).To(Equal("Hello World\n"))
			})
		})

		g.Describe("Script", func() {
			var script *Script
			g.It("puts the name in the env", func() {
				script = NewScript("test/Script", "echo $NAME")
				script.Stdout = &bytes.Buffer{}
				script.Run()
				Expect(script.Stdout.(*bytes.Buffer).String()).To(Equal("test/Script\n"))
			})
		})
	})
}
