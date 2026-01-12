package main

import (
	"context"
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	ctx := context.Background()

	cli, err := NewDockerClient()
	if err != nil {
		log.Fatalf("failed to initialize docker client: %v", err)
	}
	defer cli.Close()

	// Find auto-named containers
	candidates, err := FindAutoNamedContainers(ctx, cli)
	if err != nil {
		log.Fatalf("failed to find auto-named containers: %v", err)
	}

	if len(candidates) == 0 {
		fmt.Println("No auto-named containers found.")
		return
	}

	program := tea.NewProgram(NewCleanupModel(ctx, cli, candidates))
	if _, err := program.Run(); err != nil {
		log.Fatalf("failed to start TUI: %v", err)
	}
}
