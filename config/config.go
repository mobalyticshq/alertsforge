package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Runbook struct {
	Description  string            `yaml:"description"`
	EnricherName string            `yaml:"enricherName"`
	Config       map[string]string `yaml:"config"`
}

type RunbooksConfig struct {
	EnrichmentFlow []EnrichmentStep `yaml:"enrichment_flow"`
	OncallMessage  `yaml:"oncall_message"`
	Silences       []Silence `yaml:"silenced_alerts"`
}

type Silence struct {
	LabelsSelector map[string]string `yaml:"labelsSelector"`
	Explanation    string            `yaml:"explanation"`
}
type EnrichmentStep struct {
	LabelsSelector map[string]string `yaml:"labelsSelector"`
	Runbooks       []Runbook         `yaml:"runbooks"`
}

type OncallMessage struct {
	Title           string `yaml:"title,omitempty"`
	SlackMessage    string `yaml:"slack_message,omitempty"`
	WebMessage      string `yaml:"web_message,omitempty"`
	SimpleMessage   string `yaml:"simple_message,omitempty"`
	TelegramMessage string `yaml:"telegram_message,omitempty"`
	EscalationChain string `yaml:"escalation_chain,omitempty"`
}

type Config struct {
	mainConfig *RunbooksConfig
}
type ConfigInterface interface {
	LoadRunbooksConfig(filename string) error
}

func (c *Config) LoadRunbooksConfig(filename string) (*RunbooksConfig, error) {

	configLoaded, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	c.mainConfig = &RunbooksConfig{}
	err = yaml.Unmarshal(configLoaded, c.mainConfig)
	if err != nil {
		return nil, err
	}

	return c.mainConfig, nil
}
