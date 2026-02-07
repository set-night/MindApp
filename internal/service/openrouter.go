package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/set-night/mindapp/internal/config"
	"github.com/set-night/mindapp/internal/domain"
)

type OpenRouterService struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	cache      *ModelsCache
}

func NewOpenRouterService(apiKey string) *OpenRouterService {
	return &OpenRouterService{
		apiKey:     apiKey,
		baseURL:    "https://openrouter.ai/api/v1",
		httpClient: &http.Client{Timeout: config.RequestTimeout},
		cache:      NewModelsCache(config.ModelCacheDuration),
	}
}

type ChatMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature *float64      `json:"temperature,omitempty"`
}

type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int     `json:"prompt_tokens"`
		CompletionTokens int     `json:"completion_tokens"`
		TotalCost        float64 `json:"total_cost"`
	} `json:"usage"`
}

func (s *OpenRouterService) ListModels(ctx context.Context) ([]domain.AIModel, error) {
	if cached := s.cache.Get(); cached != nil {
		return cached, nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", s.baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch models: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result struct {
		Data []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Pricing     struct {
				Prompt     string `json:"prompt"`
				Completion string `json:"completion"`
			} `json:"pricing"`
			ContextLength int `json:"context_length"`
			TopProvider   struct {
				ContextLength int `json:"context_length"`
			} `json:"top_provider"`
			Architecture struct {
				Modality string `json:"modality"`
			} `json:"architecture"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse models: %w", err)
	}

	models := make([]domain.AIModel, 0, len(result.Data))
	for _, m := range result.Data {
		var promptPrice, completionPrice float64
		fmt.Sscanf(m.Pricing.Prompt, "%f", &promptPrice)
		fmt.Sscanf(m.Pricing.Completion, "%f", &completionPrice)

		// Prices from OpenRouter are per token, convert to per 1M tokens
		promptPrice *= 1_000_000
		completionPrice *= 1_000_000

		ctxLen := m.ContextLength
		if m.TopProvider.ContextLength > 0 {
			ctxLen = m.TopProvider.ContextLength
		}

		model := domain.AIModel{
			ID:              m.ID,
			Name:            m.Name,
			Description:     m.Description,
			PromptPrice:     promptPrice,
			CompletionPrice: completionPrice,
			ContextLength:   ctxLen,
			Capabilities:    detectCapabilities(m.ID, m.Architecture.Modality),
		}
		models = append(models, model)
	}

	s.cache.Set(models)
	return models, nil
}

func (s *OpenRouterService) Chat(ctx context.Context, messages []ChatMessage, model string, temperature *float64) (*ChatResponse, error) {
	// Skip temperature for Gemini models
	if strings.Contains(strings.ToLower(model), "gemini") {
		temperature = nil
	}

	chatReq := ChatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: temperature,
	}

	payload, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chat request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limited by OpenRouter (429)")
	}
	if resp.StatusCode == http.StatusServiceUnavailable {
		return nil, fmt.Errorf("OpenRouter service unavailable (503)")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &chatResp, nil
}

func (s *OpenRouterService) GetModel(ctx context.Context, modelID string) (*domain.AIModel, error) {
	models, err := s.ListModels(ctx)
	if err != nil {
		return nil, err
	}
	for _, m := range models {
		if m.ID == modelID {
			return &m, nil
		}
	}
	return nil, domain.ErrModelNotFound
}

func detectCapabilities(modelID, modality string) domain.ModelCapabilities {
	id := strings.ToLower(modelID)
	caps := domain.ModelCapabilities{}

	// Vision detection
	if strings.Contains(id, "vision") || strings.Contains(id, "gpt-4o") ||
		strings.Contains(id, "claude-3") || strings.Contains(id, "gemini") ||
		strings.Contains(id, "llava") || strings.Contains(modality, "image") {
		caps.Vision = true
	}

	// Audio detection
	if strings.Contains(id, "audio") || strings.Contains(modality, "audio") {
		caps.Audio = true
	}

	// Image generation
	if strings.Contains(id, "dall-e") || strings.Contains(id, "stable-diffusion") ||
		strings.Contains(id, "flux") || strings.Contains(id, "imagen") {
		caps.ImageGeneration = true
	}

	// File support (vision models generally support files)
	if caps.Vision {
		caps.Files = true
	}

	return caps
}
