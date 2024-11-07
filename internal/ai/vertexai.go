package ai

import (
	"context"
	"log"
	"reflect"
	"fmt"

	"cloud.google.com/go/vertexai/genai"
)

type VertexAI struct {
	context context.Context
	client  *genai.Client
	model   string
}

func (ai *VertexAI) GetResponse(prompt *Prompt) (string, error) {
	model := ai.client.GenerativeModel(ai.model)
	//Temperature recommended by LLM
	model.SetTemperature(0.5)

	resp, err := model.GenerateContent(ai.context, genai.Text(prompt.SystemPrompt + prompt.UserPrompt))
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	return getResponseText(resp), nil
}

func getResponseText(resp *genai.GenerateContentResponse) string {
	result := ""
	for _, cand := range resp.Candidates {
		for _, part := range cand.Content.Parts {
			if reflect.TypeOf(part) == reflect.TypeOf(genai.Text("")) {
				result += string(part.(genai.Text)) + "\n"
			}
		}
	}
	return result
}

func NewVertexAIClient(projectID, localtion, modelName string) (*VertexAI, error) {
	// Secret is provided in json and PATH is specified in the environment variable `GOOGLE_APPLICATION_CREDENTIALS`.
	ctx := context.Background()
	client, err := genai.NewClient(ctx, projectID, localtion)
	if err != nil {
		return nil, fmt.Errorf("new client error: %w", err)
	}
	return &VertexAI{
		context: ctx,
		client:  client,
		model:   modelName,
	}, nil
}
