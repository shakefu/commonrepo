package commonrepo

import (
	"testing"

	. "github.com/onsi/gomega"
	goblin "github.com/shakefu/goblin"
)

func TestCommon(t *testing.T) {
	g := goblin.Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) }) // Gomega hook
	g.Describe("GetLocalRepo", func() {
		g.It("should work", func() {
			repo, err := GetLocalRepo()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(repo).ShouldNot(BeNil())
		})
	})
}
