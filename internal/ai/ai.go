package ai

import "encoding/json"

type Ai interface {
	GetResponse(prompt *Prompt) (string, error)
}

type Image struct {
	Data      []byte
	Extension string
}

// StructuredOutputOptions holds options for structured output
type StructuredOutputOptions struct {
	Enabled    bool
	SchemaName string
	Schema     map[string]interface{}
}

type Prompt struct {
	UserPrompt       string
	SystemPrompt     string
	Images           []Image
	StructuredOutput *StructuredOutputOptions
}

// GetSchemaJSON returns the schema as JSON bytes
func (s *StructuredOutputOptions) GetSchemaJSON() (json.RawMessage, error) {
	if s == nil || s.Schema == nil {
		return nil, nil
	}
	return json.Marshal(s.Schema)
}
