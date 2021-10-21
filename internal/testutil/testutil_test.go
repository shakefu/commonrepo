package testutil_test

import (
	"testing"

	. "github.com/onsi/gomega"
	. "github.com/shakefu/commonrepo/internal/testutil"
	"github.com/shakefu/goblin"
)

func TestTestutil(t *testing.T) {
	g := goblin.Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) }) // Gomega hook
	g.Describe("testutil", func() {
		g.Describe("LocalRepo", func() {
			g.It("should work", func() {
				repo := LocalRepo()
				Expect(repo).ShouldNot(BeNil())
			})
		})
	})
}
