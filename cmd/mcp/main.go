package main

import (
	"flag"
	"log"
	"os"

	mcpserver "github.com/3-shake/alert-menta/internal/mcp"
	"github.com/3-shake/alert-menta/internal/utils"
)

func main() {
	configFile := flag.String("config", ".alert-menta.user.yaml", "Configuration file path")
	ghToken := flag.String("github-token", "", "GitHub token (or set GITHUB_TOKEN env)")
	aiKey := flag.String("api-key", "", "AI API key (or set OPENAI_API_KEY/ANTHROPIC_API_KEY env)")
	flag.Parse()

	// Get tokens from environment if not provided
	githubToken := *ghToken
	if githubToken == "" {
		githubToken = os.Getenv("GITHUB_TOKEN")
	}

	apiKey := *aiKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("ANTHROPIC_API_KEY")
		}
	}

	if githubToken == "" {
		log.Fatal("GitHub token is required (--github-token or GITHUB_TOKEN env)")
	}

	if apiKey == "" {
		log.Fatal("AI API key is required (--api-key or OPENAI_API_KEY/ANTHROPIC_API_KEY env)")
	}

	// Load configuration
	cfg, err := utils.NewConfig(*configFile)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Create and start MCP server
	server := mcpserver.NewServer(cfg, githubToken, apiKey)

	if err := server.ServeStdio(); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
}
