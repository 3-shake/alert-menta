package main

import (
	"flag"
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
    cfg := parseFlags()
    
    logger := initLogger()

    loadedConfig := loadConfiguration(cfg.configFile, logger)

    validateCommand(cfg.command, loadedConfig, logger)

    issue := getGitHubIssue(cfg.owner, cfg.repo, cfg.issueNumber, cfg.ghToken)

    userPrompt := constructUserPrompt(issue, loadedConfig, logger)

    prompt := constructPrompt(cfg.command, cfg.intent, userPrompt, loadedConfig, logger)

    aic := getAIClient(cfg.oaiKey, loadedConfig, logger)

    comment, err := aic.GetResponse(prompt)
    if err != nil {
        logger.Fatalf("Error getting Response: %v", err)
    }
    logger.Println("Response:", comment)
	
    if err := issue.PostComment(comment); err != nil {
        logger.Fatalf("Error creating comment: %v", err)
    }
}

// Parse command-line flags
func parseFlags() *Config {
    repo := flag.String("repo", "", "Repository name")
    owner := flag.String("owner", "", "Repository owner")
    issueNumber := flag.Int("issue", 0, "Issue number")
    intent := flag.String("intent", "", "Question or intent for the 'ask' command")
    command := flag.String("command", "", "Commands to be executed by AI.")
    configFile := flag.String("config", "", "Configuration file")
    ghToken := flag.String("github-token", "", "GitHub token")
    oaiKey := flag.String("api-key", "", "OpenAI api key")
    flag.Parse()
    if *repo == "" || *owner == "" || *issueNumber == 0 || *ghToken == "" || *command == "" || *configFile == "" {
        flag.PrintDefaults()
        os.Exit(1)
    }
    return &Config{
        repo:        *repo,
        owner:       *owner,
        issueNumber: *issueNumber,
        intent:      *intent,
        command:     *command,
        configFile:  *configFile,
        ghToken:     *ghToken,
        oaiKey:      *oaiKey,
    }
}

// Initialize a logger
func initLogger() *log.Logger {
    return log.New(
        os.Stdout, "[alert-menta main] ",
        log.Ldate|log.Ltime|log.Llongfile|log.Lmsgprefix,
    )
}

// Load and validate configuration
func loadConfiguration(configFile string, logger *log.Logger) *utils.Config {
    cfg, err := utils.NewConfig(configFile)
    if err != nil {
        logger.Fatalf("Error loading config: %v", err)
    }
    return cfg
}

// Validate the provided command
func validateCommand(command string, cfg *utils.Config, logger *log.Logger) {
    if _, ok := cfg.Ai.Commands[command]; !ok {
        allowedCommands := make([]string, 0, len(cfg.Ai.Commands))
        for cmd := range cfg.Ai.Commands {
            allowedCommands = append(allowedCommands, cmd)
        }
        logger.Fatalf("Invalid command: %s. Allowed commands are %s.", command, strings.Join(allowedCommands, ", "))
    }
}

// Get GitHub issue instance
func getGitHubIssue(owner, repo string, issueNumber int, ghToken string) *github.GitHubIssue {
    return github.NewIssue(owner, repo, issueNumber, ghToken)
}

// Construct user prompt from issue
func constructUserPrompt(issue *github.GitHubIssue, cfg *utils.Config, logger *log.Logger) string {
    title, err := issue.GetTitle()
    if err != nil {
        logger.Fatalf("Error getting Title: %v", err)
    }

    body, err := issue.GetBody()
    if err != nil {
        logger.Fatalf("Error getting Body: %v", err)
    }

    var userPrompt strings.Builder
    userPrompt.WriteString("Title:" + *title + "\n")
    userPrompt.WriteString("Body:" + *body + "\n")

    comments, err := issue.GetComments()
    if err != nil {
        logger.Fatalf("Error getting comments: %v", err)
    }
    for _, v := range comments {
        if *v.User.Login == "github-actions[bot]" {
            continue
        }
        if cfg.System.Debug.Log_level == "debug" {
            logger.Printf("%s: %s", *v.User.Login, *v.Body)
        }
        userPrompt.WriteString(*v.User.Login + ":" + *v.Body + "\n")
    }
    return userPrompt.String()
}

// Construct AI prompt
func constructPrompt(command, intent, userPrompt string, cfg *utils.Config, logger *log.Logger) ai.Prompt {
    var systemPrompt string
    if command == "ask" {
        if intent == "" {
            logger.Fatalf("Error: intent is required for 'ask' command")
        }
        systemPrompt = cfg.Ai.Commands[command].System_prompt + intent + "\n"
    } else {
        systemPrompt = cfg.Ai.Commands[command].System_prompt
    }
    logger.Println("\x1b[34mPrompt: |\n", systemPrompt, userPrompt, "\x1b[0m")
    return ai.Prompt{UserPrompt: userPrompt, SystemPrompt: systemPrompt}
}

// Initialize AI client
func getAIClient(oaiKey string, cfg *utils.Config, logger *log.Logger) ai.Ai {
    switch cfg.Ai.Provider {
    case "openai":
        if oaiKey == "" {
            logger.Fatalf("Error: Please provide your Open AI API key.")
        }
        logger.Println("Using OpenAI API")
        logger.Println("OpenAI model:", cfg.Ai.OpenAI.Model)
        return ai.NewOpenAIClient(oaiKey, cfg.Ai.OpenAI.Model)
    case "vertexai":
        logger.Println("Using VertexAI API")
        logger.Println("VertexAI model:", cfg.Ai.VertexAI.Model)
        return ai.NewVertexAIClient(cfg.Ai.VertexAI.Project, cfg.Ai.VertexAI.Region, cfg.Ai.VertexAI.Model)
    default:
        logger.Fatalf("Error: Invalid provider")
        return nil
    }
}