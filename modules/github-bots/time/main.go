package main

import (
	"log"

	"github.com/chainguard-dev/terraform-infra-common/modules/github-bots/sdk"
)

func main() {
	if err := sdk.Serve(New()); err != nil {
		log.Fatal(err)
	}
}
