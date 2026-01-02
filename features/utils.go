package features

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Secret represents a single secret in the ECS task definition, with a name and SSM ARN.
type Secret struct {
	Name      string `json:"name"`      // The environment variable name in the container.
	ValueFrom string `json:"valueFrom"` // The SSM parameter ARN.
}

// Environment represents a static environment variable in the ECS task definition.
type Environment struct {
	Name  string `json:"name"`  // The environment variable name.
	Value string `json:"value"` // The static value.
}

// ExtendedSecret extends Secret with type and value for pusher functionality.
type ExtendedSecret struct {
	Name      string        `json:"name"`            // The environment variable name.
	ValueFrom string        `json:"valueFrom"`       // The SSM parameter ARN.
	Type      ParameterType `json:"type,omitempty"`  // Parameter type: string, stringlist, securestring.
	Value     string        `json:"value,omitempty"` // The value to store in SSM.
}

// ContainerDefinition holds the environment and secrets arrays for a container.
type ContainerDefinition struct {
	Environment []Environment    `json:"environment"` // Static environment variables.
	Secrets     []ExtendedSecret `json:"secrets"`     // Secrets with extended fields for pusher.
}

// TaskDefinition is the top-level structure for parsing the ECS task definition JSON.
type TaskDefinition struct {
	ContainerDefinitions []ContainerDefinition `json:"containerDefinitions"` // List of containers in the task.
}

// Config holds configuration settings for the tool.
type Config struct {
	ParameterPrefix string `json:"parameterPrefix"` // Prefix for parameter paths, e.g., "/preprod/testing/"
	Region          string `json:"region"`          // Default AWS region.
}

// ParameterType represents the type of SSM parameter.
type ParameterType string

const (
	StringType       ParameterType = "String"
	SecureStringType ParameterType = "SecureString"
	StringListType   ParameterType = "StringList"
)

// LoadConfig reads config.json if it exists, otherwise creates it with defaults.
func LoadConfig() (*Config, error) {
	config := &Config{
		ParameterPrefix: "/preprod/testing/",
		Region:          "ap-southeast-3",
	}
	data, err := os.ReadFile("config.json")
	if err != nil {
		// File doesn't exist, create it with defaults
		defaultData, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal default config: %w", err)
		}
		err = os.WriteFile("config.json", defaultData, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to write default config.json: %w", err)
		}
		fmt.Println("Generated default config.json")
		return config, nil
	}
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config.json: %w", err)
	}
	return config, nil
}

// extractParameterName parses an SSM parameter ARN and returns the parameter path.
// Example: arn:aws:ssm:region:account:parameter/path/name -> /path/name
func ExtractParameterName(arn string) string {
	// Split the ARN by colons to extract components.
	parts := strings.Split(arn, ":")
	if len(parts) < 6 || parts[2] != "ssm" {
		return "" // Invalid ARN format.
	}
	paramPath := parts[5] // The part after the fifth colon, e.g., parameter/path/name
	if !strings.HasPrefix(paramPath, "parameter/") {
		return "" // Not an SSM parameter ARN.
	}
	// Return the path with a leading slash.
	return "/" + strings.TrimPrefix(paramPath, "parameter/")
}
