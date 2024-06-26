package config

import (
	"fmt"
	"io/ioutil"
	"os"
)

// LoadPrompts loads prompts from files, using default prompts if the files don't exist.
func LoadPrompts() (string, string, string, string, error) {
	systemPrompt, err := loadPromptFromFile(SystemPromptFile)
	if err != nil {
		return "", "", "", "", fmt.Errorf("error loading system prompt: %w", err)
	}

	processQueryPrompt, err := loadPromptFromFile(ProcessQueryPromptFile)
	if err != nil {
		return "", "", "", "", fmt.Errorf("error loading process query prompt: %w", err)
	}

	metricSynonymPrompt, err := loadPromptFromFile(MetricSynonymPromptFile)
	if err != nil {
		return "", "", "", "", fmt.Errorf("error loading metric synonym prompt: %w", err)
	}

	labelSynonymPrompt, err := loadPromptFromFile(LabelSynonymPromptFile)
	if err != nil {
		return "", "", "", "", fmt.Errorf("error loading label synonym prompt: %w", err)
	}

	return systemPrompt, processQueryPrompt, metricSynonymPrompt, labelSynonymPrompt, nil
}

// loadPromptFromFile loads a prompt from the specified file path.
// If the file doesn't exist, it returns an empty string.
func loadPromptFromFile(filePath string) (string, error) {
	if _, err := os.Stat(filePath); err == nil {
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("error reading prompt file: %w", err)
		}
		return string(content), nil
	}
	return "", nil // Return an empty string if the file doesn't exist
}
