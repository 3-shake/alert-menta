package ai

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type Anthropic struct {
	apiKey string
	model  string
}

func (a *Anthropic) GetResponse(prompt *Prompt) (string, error) {
	// Create a new Anthropic client
	client := anthropic.NewClient(
		option.WithAPIKey(a.apiKey),
	)

	// Build content blocks for the user message
	var contentBlocks []anthropic.ContentBlockParamUnion

	// Add images first if present
	for _, image := range prompt.Images {
		mediaType := getMediaType(image.Extension)
		base64Data := base64.StdEncoding.EncodeToString(image.Data)

		imageBlock := anthropic.NewImageBlockBase64(string(mediaType), base64Data)
		contentBlocks = append(contentBlocks, imageBlock)
	}

	// Add text content
	contentBlocks = append(contentBlocks, anthropic.NewTextBlock(prompt.UserPrompt))

	// Create the message
	message, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.Model(a.model),
		MaxTokens: 4096,
		System: []anthropic.TextBlockParam{
			{
				Type: "text",
				Text: prompt.SystemPrompt,
			},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(contentBlocks...),
		},
	})
	if err != nil {
		return "", fmt.Errorf("Anthropic API error: %w", err)
	}

	// Extract text from response
	var response string
	for _, block := range message.Content {
		if block.Type == "text" {
			response += block.Text
		}
	}

	return response, nil
}

func NewAnthropicClient(apiKey string, model string) *Anthropic {
	return &Anthropic{
		apiKey: apiKey,
		model:  model,
	}
}

// getMediaType converts file extension to MIME type
func getMediaType(ext string) anthropic.Base64ImageSourceMediaType {
	switch ext {
	case "png":
		return anthropic.Base64ImageSourceMediaTypeImagePNG
	case "gif":
		return anthropic.Base64ImageSourceMediaTypeImageGIF
	case "webp":
		return anthropic.Base64ImageSourceMediaTypeImageWebP
	default:
		return anthropic.Base64ImageSourceMediaTypeImageJPEG
	}
}
