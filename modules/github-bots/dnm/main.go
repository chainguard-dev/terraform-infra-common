package main

import (
	"github.com/chainguard-dev/terraform-infra-common/modules/github-bots/sdk"
)

func main() { sdk.Serve(New()) }
