package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/3-shake/alert-menta/internal/utils"
	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

type OpenAI struct {
	apiKey string
	model  string
}

func (ai *OpenAI) GetResponse(prompt *Prompt) (string, error) {
	// Create a new OpenAI client
	keyCredential := azcore.NewKeyCredential(ai.apiKey)
	client, err := azopenai.NewClientForOpenAI("https://api.openai.com/v1/", keyCredential, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	// Convert images to base64
	base64Images := func(images []Image) []string {
		var base64Images []string
		for _, image := range images {
			base64Images = append(base64Images, utils.ImageToBase64(image.Data, image.Extension))
		}
		return base64Images
	}(prompt.Images)

	// create a user prompt with text and images
	userPrompt := []azopenai.ChatCompletionRequestMessageContentPartClassification{
		&azopenai.ChatCompletionRequestMessageContentPartText{Text: &prompt.UserPrompt},
	}
	for _, image := range base64Images {
		userPrompt = append(userPrompt, &azopenai.ChatCompletionRequestMessageContentPartImage{ImageURL: &azopenai.ChatCompletionRequestMessageContentPartImageURL{URL: &image}})
	}

	// Create a chat request with the prompt
	messages := []azopenai.ChatRequestMessageClassification{
		&azopenai.ChatRequestSystemMessage{
			Content: azopenai.NewChatRequestSystemMessageContent(prompt.SystemPrompt),
		},
		&azopenai.ChatRequestUserMessage{
			Content: azopenai.NewChatRequestUserMessageContent(userPrompt),
		},
	}

	// Build chat completion options
	options := azopenai.ChatCompletionsOptions{
		DeploymentName: &ai.model,
		Messages:       messages,
	}

	// Add structured output (JSON mode) if enabled
	if prompt.StructuredOutput != nil && prompt.StructuredOutput.Enabled {
		options.ResponseFormat = &azopenai.ChatCompletionsJSONResponseFormat{}
	}

	// Call the chat completion endpoint
	resp, err := client.GetChatCompletions(context.TODO(), options, nil)
	if err != nil {
		return "", fmt.Errorf("ChatCompletion error: %w", err)
	}

	response := *resp.Choices[0].Message.Content

	// Validate JSON output if structured output is enabled
	if prompt.StructuredOutput != nil && prompt.StructuredOutput.Enabled {
		if !json.Valid([]byte(response)) {
			return "", fmt.Errorf("structured output validation failed: response is not valid JSON")
		}
	}

	return response, nil
}

func NewOpenAIClient(apiKey string, model string) *OpenAI {
	// Specifying the model to use
	return &OpenAI{
		apiKey: apiKey,
		model:  model,
	}
}
