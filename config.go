package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v3"
)

func LoadConfig(path string) (*Config, error) {
	// If file doesn't exist, try loading from environment variables or use defaults
	data, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file %s not found. Please create it from config.example.yml", path)
		}
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Override with environment variables if present
	if envKey := os.Getenv("INTERVALS_API_KEY"); envKey != "" {
		config.APIKey = envKey
	}
	if envID := os.Getenv("INTERVALS_ATHLETE_ID"); envID != "" {
		config.AthleteID = envID
	}

	if config.APIKey == "" || config.AthleteID == "" {
		return nil, fmt.Errorf("API key or Athlete ID missing from config and environment")
	}

	return &config, nil
}
