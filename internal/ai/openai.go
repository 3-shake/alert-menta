package ai

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

type OpenAI struct {
	apiKey string
	model  string
}

func (ai *OpenAI) GetResponse(prompt string) (string, error) {
	// Create a new OpenAI client
	keyCredential := azcore.NewKeyCredential(ai.apiKey)
	client, _ := azopenai.NewClientForOpenAI("https://api.openai.com/v1/", keyCredential, nil)

	// Create a chat request with the prompt
	messages := []azopenai.ChatRequestMessageClassification{
		&azopenai.ChatRequestUserMessage{
			Content: azopenai.NewChatRequestUserMessageContent(prompt),
		},
	}

	// Call the chat completion endpoint
	resp, err := client.GetChatCompletions(context.TODO(), azopenai.ChatCompletionsOptions{
		DeploymentName: &ai.model,
		Messages:       messages,
	}, nil)
	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return "", err
	}

	// Print the response
	// resp.Choices[0].Message.Content is type *string with azopenai and type string with go-openai
	fmt.Println(*resp.Choices[0].Message.Content)

	return *resp.Choices[0].Message.Content, nil
}

func NewOpenAIClient(apiKey string, model string) *OpenAI {
	// Set the OpenAI API key (read from the environment variable)
	if apiKey == "" {
		fmt.Println("Error: OPENAI_API_KEY environment variable not set.")
		return nil
	}
	// Specifying the model to use
	return &OpenAI{
		apiKey: apiKey,
		model:  model,
	}
}
