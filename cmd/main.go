package main

import (
	"flag"
	"log"
	"os"
	"strings"
    "fmt"

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

    err = validateCommand(cfg.command, loadedcfg)
    if err != nil {
        logger.Fatalf("Error validating command: %v", err)
    }

    issue := github.NewIssue(cfg.owner, cfg.repo, cfg.issueNumber, cfg.ghToken)

    userPrompt, err := constructUserPrompt(issue, loadedcfg, logger)
    if err != nil {
        logger.Fatalf("Erro constructing userPrompt: %v", err)
    }

    prompt, err := constructPrompt(cfg.command, cfg.intent, userPrompt, loadedcfg, logger)
    if err != nil {
        logger.Fatalf("Error constructing prompt: %v", err)
    }

    aic, err := getAIClient(cfg.oaiKey, loadedcfg, logger)
    if err != nil {
        logger.Fatalf("Error geting AI client: %v", err)
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
        return fmt.Errorf("Invalid command: %s. Allowed commands are %s", command, strings.Join(allowedCommands, ", "))
    }
    return nil
}

// Construct user prompt from issue
func constructUserPrompt(issue *github.GitHubIssue, cfg *utils.Config, logger *log.Logger) (string, error) {
    title, err := issue.GetTitle()
    if err != nil {
        return "", fmt.Errorf("Error getting Title: %w", err)
    }

    body, err := issue.GetBody()
    if err != nil {
        return "", fmt.Errorf("Error getting Body: %w", err)
    }

    var userPrompt strings.Builder
    userPrompt.WriteString("Title:" + *title + "\n")
    userPrompt.WriteString("Body:" + *body + "\n")

    comments, err := issue.GetComments()
    if err != nil {
        return "", fmt.Errorf("Error getting comments: %w", err)
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
    return userPrompt.String(), nil
}

// Construct AI prompt
func constructPrompt(command, intent, userPrompt string, cfg *utils.Config, logger *log.Logger) (*ai.Prompt, error){
    var systemPrompt string
    if command == "ask" {
        if intent == "" {
            return nil, fmt.Errorf("Error: intent is required for 'ask' command")
        }
        systemPrompt = cfg.Ai.Commands[command].System_prompt + intent + "\n"
    } else {
        systemPrompt = cfg.Ai.Commands[command].System_prompt
    }
    logger.Println("\x1b[34mPrompt: |\n", systemPrompt, userPrompt, "\x1b[0m")
    return &ai.Prompt{UserPrompt: userPrompt, SystemPrompt: systemPrompt}, nil
}

// Initialize AI client
func getAIClient(oaiKey string, cfg *utils.Config, logger *log.Logger) (ai.Ai, error) {
    switch cfg.Ai.Provider {
    case "openai":
        if oaiKey == "" {
            return nil, fmt.Errorf("Error: Please provide your Open AI API key")
        }
        logger.Println("Using OpenAI API")
        logger.Println("OpenAI model:", cfg.Ai.OpenAI.Model)
        return ai.NewOpenAIClient(oaiKey, cfg.Ai.OpenAI.Model), nil
    case "vertexai":
        logger.Println("Using VertexAI API")
        logger.Println("VertexAI model:", cfg.Ai.VertexAI.Model)
        aic, err := ai.NewVertexAIClient(cfg.Ai.VertexAI.Project, cfg.Ai.VertexAI.Region, cfg.Ai.VertexAI.Model)
        if err != nil {
            return nil, fmt.Errorf("Error: new Vertex AI client: %w", err)
        }
        return aic, nil
    default:
        return nil, fmt.Errorf("Error: Invalid provider")
    }
}