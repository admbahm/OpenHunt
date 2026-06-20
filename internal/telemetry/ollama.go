package telemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// OllamaClient handles communication with a local Ollama instance.
type OllamaClient struct {
	Endpoint string
	Model    string
	HTTP     *http.Client
}

// NewOllamaClient initializes a new Ollama client.
func NewOllamaClient(endpoint, model string) *OllamaClient {
	if endpoint == "" {
		endpoint = "http://localhost:11434/api/generate"
	}
	if model == "" {
		model = "llama3"
	}
	return &OllamaClient{
		Endpoint: endpoint,
		Model:    model,
		HTTP: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// OllamaRequest represents the payload for the Ollama API.
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Format string `json:"format"` // "json" to enforce structured output
}

// OllamaResponse represents the response from the Ollama API.
type OllamaResponse struct {
	Response string `json:"response"`
}

// AnalyzeJob sends a job description to Ollama and returns a structured analysis.
func (c *OllamaClient) AnalyzeJob(description string) (*AnalysisResult, error) {
	prompt := fmt.Sprintf(`Analyze the following job description and return a JSON object with these fields:
- base_salary_min (integer)
- base_salary_max (integer)
- tech_stack (list of strings)
- regulatory_gates (list of strings, e.g., HIPAA, GxP, Secret Clearance)
- role_type (string, either "Individual Contributor" or "Management")

Job Description:
%s`, description)

	reqBody := OllamaRequest{
		Model:  c.Model,
		Prompt: prompt,
		Stream: false,
		Format: "json",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.HTTP.Post(c.Endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status: %d", resp.StatusCode)
	}

	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode ollama response: %w", err)
	}

	var result AnalysisResult
	if err := json.Unmarshal([]byte(ollamaResp.Response), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal analysis result: %w", err)
	}

	return &result, nil
}
