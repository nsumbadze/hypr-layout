package main

import (
	"os"

	"github.com/nsumbadze/hypr-layout/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:]))
}
