package main

import (
	"fmt"

	"github.com/shakefu/commonrepo/memrepo"
	"github.com/shakefu/commonrepo/runner"
)

func main() {
	fmt.Println("Here we go")
	repo, _ := memrepo.NewRepo("git@github.com:shakefu/home.git")
	runner.Run("install/darwin_amd64/00-curl", repo)
}
