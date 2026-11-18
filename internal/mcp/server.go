package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/3-shake/alert-menta/internal/ai"
	"github.com/3-shake/alert-menta/internal/github"
	"github.com/3-shake/alert-menta/internal/utils"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps the MCP server with alert-menta functionality
type Server struct {
	mcpServer   *server.MCPServer
	config      *utils.Config
	githubToken string
	aiKey       string
}

// NewServer creates a new MCP server for alert-menta
func NewServer(config *utils.Config, githubToken, aiKey string) *Server {
	s := &Server{
		config:      config,
		githubToken: githubToken,
		aiKey:       aiKey,
	}

	mcpServer := server.NewMCPServer(
		"alert-menta",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Register tools
	s.registerTools(mcpServer)
	s.mcpServer = mcpServer

	return s
}

// registerTools adds all alert-menta tools to the MCP server
func (s *Server) registerTools(mcpServer *server.MCPServer) {
	// get_incident tool
	mcpServer.AddTool(
		mcp.NewTool("get_incident",
			mcp.WithDescription("Get incident information from a GitHub Issue"),
			mcp.WithString("owner", mcp.Required(), mcp.Description("Repository owner")),
			mcp.WithString("repo", mcp.Required(), mcp.Description("Repository name")),
			mcp.WithNumber("issue_number", mcp.Required(), mcp.Description("Issue number")),
		),
		s.handleGetIncident,
	)

	// analyze_incident tool
	mcpServer.AddTool(
		mcp.NewTool("analyze_incident",
			mcp.WithDescription("Analyze incident using AI (describe, suggest, analysis, postmortem, runbook, timeline)"),
			mcp.WithString("owner", mcp.Required(), mcp.Description("Repository owner")),
			mcp.WithString("repo", mcp.Required(), mcp.Description("Repository name")),
			mcp.WithNumber("issue_number", mcp.Required(), mcp.Description("Issue number")),
			mcp.WithString("command", mcp.Required(),
				mcp.Description("Analysis command: describe, suggest, analysis, postmortem, runbook, timeline"),
			),
			mcp.WithString("intent", mcp.Description("Additional intent for ask command")),
		),
		s.handleAnalyzeIncident,
	)

	// post_comment tool
	mcpServer.AddTool(
		mcp.NewTool("post_comment",
			mcp.WithDescription("Post a comment to a GitHub Issue"),
			mcp.WithString("owner", mcp.Required(), mcp.Description("Repository owner")),
			mcp.WithString("repo", mcp.Required(), mcp.Description("Repository name")),
			mcp.WithNumber("issue_number", mcp.Required(), mcp.Description("Issue number")),
			mcp.WithString("body", mcp.Required(), mcp.Description("Comment body in Markdown")),
		),
		s.handlePostComment,
	)

	// list_commands tool
	mcpServer.AddTool(
		mcp.NewTool("list_commands",
			mcp.WithDescription("List all available alert-menta commands"),
		),
		s.handleListCommands,
	)
}

// handleGetIncident retrieves incident information from GitHub Issue
func (s *Server) handleGetIncident(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	owner, _ := args["owner"].(string)
	repo, _ := args["repo"].(string)
	issueNum, _ := args["issue_number"].(float64)

	issue := github.NewIssue(owner, repo, int(issueNum), s.githubToken)

	title, err := issue.GetTitle()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get issue title: %v", err)), nil
	}

	body, err := issue.GetBody()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get issue body: %v", err)), nil
	}

	comments, err := issue.GetComments()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get comments: %v", err)), nil
	}

	// Format response
	result := fmt.Sprintf("## Issue #%d: %s\n\n", int(issueNum), *title)
	result += fmt.Sprintf("**Body:**\n%s\n\n", *body)
	result += fmt.Sprintf("**Comments (%d):**\n", len(comments))
	for i, c := range comments {
		if c.User != nil && c.Body != nil {
			result += fmt.Sprintf("\n### Comment %d by @%s\n%s\n", i+1, *c.User.Login, *c.Body)
		}
	}

	return mcp.NewToolResultText(result), nil
}

// handleAnalyzeIncident runs an analysis command on the incident
func (s *Server) handleAnalyzeIncident(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	owner, _ := args["owner"].(string)
	repo, _ := args["repo"].(string)
	issueNum, _ := args["issue_number"].(float64)
	command, _ := args["command"].(string)
	intent, _ := args["intent"].(string)

	// Validate command
	cmdConfig, ok := s.config.Ai.Commands[command]
	if !ok {
		availableCommands := make([]string, 0, len(s.config.Ai.Commands))
		for cmd := range s.config.Ai.Commands {
			availableCommands = append(availableCommands, cmd)
		}
		return mcp.NewToolResultError(fmt.Sprintf("Invalid command: %s. Available: %v", command, availableCommands)), nil
	}

	// Check if intent is required
	if cmdConfig.RequireIntent && intent == "" {
		return mcp.NewToolResultError(fmt.Sprintf("Command '%s' requires an intent parameter", command)), nil
	}

	// Get issue information
	issue := github.NewIssue(owner, repo, int(issueNum), s.githubToken)

	title, err := issue.GetTitle()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get issue: %v", err)), nil
	}

	body, err := issue.GetBody()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get issue body: %v", err)), nil
	}

	comments, err := issue.GetComments()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get comments: %v", err)), nil
	}

	// Build user prompt
	userPrompt := fmt.Sprintf("Title: %s\nBody: %s\n", *title, *body)
	for _, c := range comments {
		if c.User != nil && c.Body != nil && *c.User.Login != "github-actions[bot]" {
			userPrompt += fmt.Sprintf("%s: %s\n", *c.User.Login, *c.Body)
		}
	}

	// Build system prompt
	systemPrompt := cmdConfig.SystemPrompt
	if cmdConfig.RequireIntent && intent != "" {
		systemPrompt = systemPrompt + intent + "\n"
	}

	// Get AI client
	aiClient, err := s.getAIClient()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create AI client: %v", err)), nil
	}

	// Get response
	prompt := &ai.Prompt{
		UserPrompt:   userPrompt,
		SystemPrompt: systemPrompt,
	}

	response, err := aiClient.GetResponse(prompt)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("AI error: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}

// handlePostComment posts a comment to the GitHub Issue
func (s *Server) handlePostComment(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	owner, _ := args["owner"].(string)
	repo, _ := args["repo"].(string)
	issueNum, _ := args["issue_number"].(float64)
	body, _ := args["body"].(string)

	issue := github.NewIssue(owner, repo, int(issueNum), s.githubToken)

	if err := issue.PostComment(body); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to post comment: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully posted comment to %s/%s#%d", owner, repo, int(issueNum))), nil
}

// handleListCommands lists all available commands
func (s *Server) handleListCommands(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	commands := make(map[string]string)
	for cmd, config := range s.config.Ai.Commands {
		commands[cmd] = config.Description
	}

	result, _ := json.MarshalIndent(commands, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("Available commands:\n%s", string(result))), nil
}

// getAIClient creates an AI client based on configuration
func (s *Server) getAIClient() (ai.Ai, error) {
	// Check if fallback is enabled
	if s.config.Ai.Fallback.Enabled && len(s.config.Ai.Fallback.Providers) > 0 {
		return s.getAIClientWithFallback()
	}

	return s.getSingleAIClient(s.config.Ai.Provider)
}

// getSingleAIClient creates a single AI client for the given provider
func (s *Server) getSingleAIClient(provider string) (ai.Ai, error) {
	switch provider {
	case "openai":
		if s.aiKey == "" {
			return nil, fmt.Errorf("OpenAI API key is required")
		}
		return ai.NewOpenAIClient(s.aiKey, s.config.Ai.OpenAI.Model), nil
	case "anthropic":
		if s.aiKey == "" {
			return nil, fmt.Errorf("anthropic API key is required")
		}
		return ai.NewAnthropicClient(s.aiKey, s.config.Ai.Anthropic.Model), nil
	case "vertexai":
		return ai.NewVertexAIClient(s.config.Ai.VertexAI.Project, s.config.Ai.VertexAI.Region, s.config.Ai.VertexAI.Model)
	default:
		return nil, fmt.Errorf("invalid provider: %s", provider)
	}
}

// getAIClientWithFallback creates a fallback client with multiple providers
func (s *Server) getAIClientWithFallback() (ai.Ai, error) {
	var clients []ai.Ai
	var names []string

	for _, provider := range s.config.Ai.Fallback.Providers {
		client, err := s.getSingleAIClient(provider)
		if err != nil {
			continue
		}
		clients = append(clients, client)
		names = append(names, provider)
	}

	if len(clients) == 0 {
		return nil, fmt.Errorf("no valid providers configured for fallback")
	}

	fallbackConfig := ai.FallbackClientConfig{
		MaxRetries: s.config.Ai.Fallback.Retry.MaxRetries,
		DelayMs:    s.config.Ai.Fallback.Retry.DelayMs,
		Logger:     nil, // MCP server doesn't use logger
	}

	return ai.NewFallbackClient(clients, names, fallbackConfig), nil
}

// ServeStdio starts the MCP server using stdio transport
func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.mcpServer)
}
