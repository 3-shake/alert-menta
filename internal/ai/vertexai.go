package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"cloud.google.com/go/vertexai/genai"
)

type VertexAI struct {
	context context.Context
	client  *genai.Client
	model   string
}

func (ai *VertexAI) GetResponse(prompt *Prompt) (string, error) {
	model := ai.client.GenerativeModel(ai.model)
	// Temperature recommended by LLM
	model.SetTemperature(0.5)

	// Set JSON mode for structured output
	if prompt.StructuredOutput != nil && prompt.StructuredOutput.Enabled {
		model.ResponseMIMEType = "application/json"
	}

	integratedPrompt := []genai.Part{} // image + text prompt
	for _, image := range prompt.Images {
		integratedPrompt = append(integratedPrompt, genai.ImageData(image.Extension, image.Data))
	}

	// Modify prompt for structured output
	systemPrompt := prompt.SystemPrompt
	if prompt.StructuredOutput != nil && prompt.StructuredOutput.Enabled {
		schemaJSON, err := prompt.StructuredOutput.GetSchemaJSON()
		if err == nil && schemaJSON != nil {
			systemPrompt += fmt.Sprintf("\n\nRespond with valid JSON that conforms to this schema:\n%s", string(schemaJSON))
		}
	}

	integratedPrompt = append(integratedPrompt, genai.Text(systemPrompt+prompt.UserPrompt))

	// Generate AI response
	resp, err := model.GenerateContent(ai.context, integratedPrompt...)
	if err != nil {
		return "", fmt.Errorf("GenerateContent error: %w", err)
	}

	response := getResponseText(resp)

	// Validate JSON output if structured output is enabled
	if prompt.StructuredOutput != nil && prompt.StructuredOutput.Enabled {
		if !json.Valid([]byte(response)) {
			return "", fmt.Errorf("structured output validation failed: response is not valid JSON")
		}
	}

	return response, nil
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
	// If you are using gcloud cli authentication or workload identity federation, you do not need to specify the secret json file.
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
