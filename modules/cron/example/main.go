package main

import (
	"log"
	"os"
	"time"
)

func main() { log.Println("hello the time is now", time.Now(), os.Getenv("EXAMPLE_ENV")) }
