package config

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"path/filepath"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Theme string `yaml:"theme"`
}

func LoadConfig() *Config {

	var configPath string

	switch runtime.GOOS {
	case "windows":
		configPath = "%LOCALAPPDATA%\\awstui\\config.yml"
	case "darwin":
		configPath = "~/Library/Application\\ Support/awstui/config.yml"
	case "linux":
		configPath, _ = expandPath("~/.config/awstui/config.yml")

	}

	config := &Config{Theme: "tokyo_night"}
	yamlFile, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
		return config
	}
	err = yaml.Unmarshal(yamlFile, config)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	return config
}

func expandPath(path string) (string, error) {
	// 1. Expand environment variables
	expanded := os.ExpandEnv(path)

	// 2. Expand user home directory (~)
	if strings.HasPrefix(expanded, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not get user home directory: %w", err)
		}
		expanded = strings.Replace(expanded, "~", homeDir, 1)
	}

	// 3. Get the absolute path
	absPath, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Errorf("could not get absolute path for '%s': %w", expanded, err)
	}

	return absPath, nil
}
