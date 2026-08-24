package main

import (
	"fmt"
	"os"

	"github.com/Opperiesen/podman-console/internal/app"
	"github.com/Opperiesen/podman-console/internal/config"
	"github.com/Opperiesen/podman-console/internal/podman"
)

// version is a variable so release builds can override it with -ldflags -X.
var version = "0.1.0"

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("podman-console %s\n", version)
		return
	}
	store, err := config.NewStore("podman-console")
	if err != nil {
		fmt.Fprintf(os.Stderr, "podman-console: %v\n", err)
		os.Exit(1)
	}
	program := app.NewProgram(store, podman.BindingsFactory{})
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "podman-console: %v\n", err)
		os.Exit(1)
	}
}
