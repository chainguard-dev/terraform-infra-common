/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"log"
	"os"
	"time"
)

func main() { log.Println("hello the time is now", time.Now(), os.Getenv("EXAMPLE_ENV")) }
