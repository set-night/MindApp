package domain

type AIModel struct {
	ID              string
	Name            string
	Description     string
	PromptPrice     float64 // per 1M tokens
	CompletionPrice float64 // per 1M tokens
	ContextLength   int
	UsageCount      int
	Capabilities    ModelCapabilities
}

type ModelCapabilities struct {
	Vision          bool
	Audio           bool
	ImageGeneration bool
	Files           bool
}

func (m *AIModel) IsFree() bool {
	return m.PromptPrice == 0 && m.CompletionPrice == 0
}
