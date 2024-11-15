package ai

import (
	"context"
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
	client, _ := azopenai.NewClientForOpenAI("https://api.openai.com/v1/", keyCredential, nil)

	// Convert images to base64
	base64Images := func(images []Image) []string {
		var base64Images []string
		for _, image := range images {
			base64Images = append(base64Images, utils.ImageToBase64(image.Data, image.Extension))
		}
		return base64Images
	}(prompt.Images)

	// create a user prompt with text and images
	user_prompt := []azopenai.ChatCompletionRequestMessageContentPartClassification{
		&azopenai.ChatCompletionRequestMessageContentPartText{Text: &prompt.UserPrompt},
	}
	for _, image := range base64Images {
		user_prompt = append(user_prompt, &azopenai.ChatCompletionRequestMessageContentPartImage{ImageURL: &azopenai.ChatCompletionRequestMessageContentPartImageURL{URL: &image}})
	}

	// Create a chat request with the prompt
	messages := []azopenai.ChatRequestMessageClassification{
		&azopenai.ChatRequestSystemMessage{
			Content: azopenai.NewChatRequestSystemMessageContent(prompt.SystemPrompt),
		},
		&azopenai.ChatRequestUserMessage{
			Content: azopenai.NewChatRequestUserMessageContent(user_prompt),
		},
	}

	// Call the chat completion endpoint
	resp, err := client.GetChatCompletions(context.TODO(), azopenai.ChatCompletionsOptions{
		DeploymentName: &ai.model,
		Messages:       messages,
	}, nil)
	if err != nil {
		return "", fmt.Errorf("ChatCompletion error: %w", err)
	}

	// Print the response
	// resp.Choices[0].Message.Content is type *string with azopenai and type string with go-openai
	// fmt.Println(*resp.Choices[0].Message.Content)

	return *resp.Choices[0].Message.Content, nil
}

func NewOpenAIClient(apiKey string, model string) *OpenAI {
	// Specifying the model to use
	return &OpenAI{
		apiKey: apiKey,
		model:  model,
	}
}
