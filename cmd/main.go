package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/3-shake/alert-menta/internal/ai"
	"github.com/3-shake/alert-menta/internal/github"
	"github.com/3-shake/alert-menta/internal/utils"
)

// Struct to hold the command-line arguments
type Config struct {
	repo        string
	owner       string
	issueNumber int
	intent      string
	command     string
	configFile  string
	ghToken     string
	oaiKey      string
}

func main() {
	cfg := &Config{}
	flag.StringVar(&cfg.repo, "repo", "", "Repository name")
	flag.StringVar(&cfg.owner, "owner", "", "Repository owner")
	flag.IntVar(&cfg.issueNumber, "issue", 0, "Issue number")
	flag.StringVar(&cfg.intent, "intent", "", "Question or intent for the 'ask' command")
	flag.StringVar(&cfg.command, "command", "", "Commands to be executed by AI. Commands defined in the configuration file are available.")
	flag.StringVar(&cfg.configFile, "config", "", "Configuration file")
	flag.StringVar(&cfg.ghToken, "github-token", "", "GitHub token")
	flag.StringVar(&cfg.oaiKey, "api-key", "", "OpenAI api key")
	flag.Parse()

	if cfg.repo == "" || cfg.owner == "" || cfg.issueNumber == 0 || cfg.ghToken == "" || cfg.command == "" || cfg.configFile == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	logger := log.New(
		os.Stdout, "[alert-menta main] ",
		log.Ldate|log.Ltime|log.Llongfile|log.Lmsgprefix,
	)

	loadedcfg, err := utils.NewConfig(cfg.configFile)
	if err != nil {
		logger.Fatalf("Error loading config: %v", err)
	}

	issue := github.NewIssue(cfg.owner, cfg.repo, cfg.issueNumber, cfg.ghToken)

	// Validate command
	err = validateCommand(cfg.command, loadedcfg)
	if err != nil {
		// Get available commands for the error message
		availableCommands := getAvailableCommands(loadedcfg)
		usageMessage := fmt.Sprintf("**Error**: %v\n\n**Available commands:**\n", err)

		// Add each command with its description to the usage message
		for cmd, description := range availableCommands {
			usageMessage += fmt.Sprintf("- `/%s`: %s\n", cmd, description)
		}

		// Post the usage message as a comment
		if postErr := issue.PostComment(usageMessage); postErr != nil {
			logger.Fatalf("Error posting error comment: %v", postErr)
		}

		// Exit with error code
		logger.Printf("Error validating command: %v", err)
		os.Exit(1)
	}

	// Check if intent is required for this command and missing
	needsIntent, err := commandNeedsIntent(cfg.command, loadedcfg)
	if err != nil {
		logger.Fatalf("Error checking if intent is required: %v", err)
	}
	if needsIntent && cfg.intent == "" {
		usageMessage := fmt.Sprintf("**Error**: The `/%s` command requires additional text after the command.\n\n**Usage**: `/%s [your text here]`",
			cfg.command, cfg.command)

		// Post the usage message as a comment
		if postErr := issue.PostComment(usageMessage); postErr != nil {
			logger.Fatalf("Error posting error comment: %v", postErr)
		}

		// Exit with error code
		logger.Printf("Error: intent required for command %s", cfg.command)
		os.Exit(1)
	}

	userPrompt, imgs, err := constructUserPrompt(cfg.ghToken, issue, loadedcfg, logger)
	if err != nil {
		logger.Fatalf("Erro constructing userPrompt: %v", err)
	}

	prompt, err := constructPrompt(cfg.command, cfg.intent, userPrompt, imgs, loadedcfg, logger)
	if err != nil {
		logger.Fatalf("Error constructing prompt: %v", err)
	}

	aic, err := getAIClient(cfg.oaiKey, loadedcfg, logger)
	if err != nil {
		logger.Fatalf("Error getting AI client: %v", err)
	}

	comment, err := aic.GetResponse(prompt)
	if err != nil {
		logger.Fatalf("Error getting Response: %v", err)
	}
	logger.Println("Response:", comment)

	if err := issue.PostComment(comment); err != nil {
		logger.Fatalf("Error creating comment: %v", err)
	}
}

// Validate the provided command
func validateCommand(command string, cfg *utils.Config) error {
	if _, ok := cfg.Ai.Commands[command]; !ok {
		allowedCommands := make([]string, 0, len(cfg.Ai.Commands))
		for cmd := range cfg.Ai.Commands {
			allowedCommands = append(allowedCommands, cmd)
		}
		return fmt.Errorf("invalid command: %s, allowed commands are %s", command, strings.Join(allowedCommands, ", "))
	}
	return nil
}

// Check if a command requires an intent
func commandNeedsIntent(command string, cfg *utils.Config) (bool, error) {
	// Get the command configuration
	cmd, ok := cfg.Ai.Commands[command]
	if !ok {
		return false, fmt.Errorf("command not found: %s", command)
	}

	// Check if this command requires intent
	return cmd.RequireIntent, nil
}

// Get available commands with descriptions for usage message
func getAvailableCommands(cfg *utils.Config) map[string]string {
	commands := make(map[string]string)
	for cmd, cmdConfig := range cfg.Ai.Commands {
		commands[cmd] = cmdConfig.Description
	}
	return commands
}

// Construct user prompt from issue
func constructUserPrompt(ghToken string, issue *github.GitHubIssue, cfg *utils.Config, logger *log.Logger) (string, []ai.Image, error) {
	var images []ai.Image

	title, err := issue.GetTitle()
	if err != nil {
		return "", nil, fmt.Errorf("getting title: %w", err)
	}

	body, err := issue.GetBody()
	if err != nil {
		return "", nil, fmt.Errorf("getting body: %w", err)
	}

	var userPrompt strings.Builder
	userPrompt.WriteString("Title:" + *title + "\n")
	userPrompt.WriteString("Body:" + *body + "\n")
	urls := utils.ExtractImageURLs(*body)
	for _, url := range urls {
		imgData, ext, err := utils.DownloadImage(url, ghToken)
		if err != nil {
			return "", nil, fmt.Errorf("Error downloading image: %w", err)
		}
		images = append(images, ai.Image{Data: imgData, Extension: ext})
	}

	comments, err := issue.GetComments()
	if err != nil {
		return "", nil, fmt.Errorf("getting comments: %w", err)
	}

	for _, v := range comments {
		if *v.User.Login == "github-actions[bot]" {
			continue
		}
		if cfg.System.Debug.LogLevel == "debug" {
			logger.Printf("%s: %s", *v.User.Login, *v.Body)
		}
		userPrompt.WriteString(*v.User.Login + ":" + *v.Body + "\n")

		urls := utils.ExtractImageURLs(*body)
		for _, url := range urls {
			imgData, ext, err := utils.DownloadImage(url, ghToken)
			if err != nil {
				return "", nil, fmt.Errorf("downloading image: %w", err)
			}
			images = append(images, ai.Image{Data: imgData, Extension: ext})
		}
	}
	return userPrompt.String(), images, nil
}

// Construct AI prompt
func constructPrompt(command, intent, userPrompt string, imgs []ai.Image, cfg *utils.Config, logger *log.Logger) (*ai.Prompt, error) {
	var systemPrompt string
	if cfg.Ai.Commands[command].RequireIntent {
		if intent == "" {
			return nil, fmt.Errorf("intent is required for '%s' command", command)
		}
		systemPrompt = cfg.Ai.Commands[command].SystemPrompt + intent + "\n"
	} else {
		systemPrompt = cfg.Ai.Commands[command].SystemPrompt
	}
	logger.Println("\x1b[34mPrompt: |\n", systemPrompt, userPrompt, "\x1b[0m")
	return &ai.Prompt{UserPrompt: userPrompt, SystemPrompt: systemPrompt, Images: imgs}, nil
}

// Initialize AI client
func getAIClient(oaiKey string, cfg *utils.Config, logger *log.Logger) (ai.Ai, error) {
	switch cfg.Ai.Provider {
	case "openai":
		if oaiKey == "" {
			return nil, fmt.Errorf("OpenAI API key is required")
		}
		logger.Println("Using OpenAI API")
		logger.Println("OpenAI model:", cfg.Ai.OpenAI.Model)
		return ai.NewOpenAIClient(oaiKey, cfg.Ai.OpenAI.Model), nil
	case "vertexai":
		logger.Println("Using VertexAI API")
		logger.Println("VertexAI model:", cfg.Ai.VertexAI.Model)
		aic, err := ai.NewVertexAIClient(cfg.Ai.VertexAI.Project, cfg.Ai.VertexAI.Region, cfg.Ai.VertexAI.Model)
		if err != nil {
			return nil, fmt.Errorf("new Vertex AI client: %w", err)
		}
		return aic, nil
	default:
		return nil, fmt.Errorf("invalid provider: %s", cfg.Ai.Provider)
	}
}
