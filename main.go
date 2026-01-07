package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
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

	// Show the list
	fmt.Printf("Found %d auto-named container(s):\n", len(candidates))
	for _, candidate := range candidates {
		fmt.Println("")
		fmt.Printf("  - Name : %s\n", primaryContainerName(candidate.Names))
		fmt.Printf("    Image: %s\n", candidate.Image)
	}

	// Ask for confirmation
	fmt.Print("\nDo you want to delete these containers? (yes/no): ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("failed to read input: %v", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "yes" && response != "y" {
		fmt.Println("Operation cancelled.")
		return
	}

	// Proceed with deletion
	removed, err := CleanAutoNamedContainers(ctx, cli)
	if err != nil {
		log.Printf("cleanup completed with errors: %v", err)
	}

	fmt.Fprintf(os.Stdout, "\nRemoved %d auto-named container(s)\n", len(removed))
	for _, name := range removed {
		fmt.Printf("  - %s\n", name)
	}
}
