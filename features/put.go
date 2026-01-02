package features

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// putParametersFromTemplate reads a custom task definition template and puts parameters to SSM.
// Handles secrets (with type/value) from the template.
func PutParametersFromTemplate(client *ssm.Client, filename string) error {
	// Read the JSON file.
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal into TaskDefinition.
	var taskDef TaskDefinition
	if err := json.Unmarshal(data, &taskDef); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	if len(taskDef.ContainerDefinitions) == 0 {
		return fmt.Errorf("no container definitions found")
	}

	container := taskDef.ContainerDefinitions[0]

	// Process secrets (push with specified type).
	for _, secret := range container.Secrets {
		if secret.Value == "" {
			log.Printf("Skipping %s: missing value", secret.Name)
			continue
		}
		paramTypeStr := strings.ToLower(string(secret.Type))
		var paramType ParameterType
		switch paramTypeStr {
		case "string":
			paramType = StringType
		case "stringlist":
			paramType = StringListType
		case "securestring":
			paramType = SecureStringType
		default:
			paramType = StringType // Default.
		}
		paramName := ExtractParameterName(secret.ValueFrom)
		if paramName == "" {
			paramName = "/preprod/testing/" + strings.ToLower(secret.Name) // Fallback.
		}
		err := PutParameter(client, paramName, secret.Value, paramType)
		if err != nil {
			return fmt.Errorf("failed to put secret %s: %w", secret.Name, err)
		}
		fmt.Printf("Put secret %s as %s\n", paramName, paramType)
	}
	return nil
}

// PutParameter stores or updates a parameter in AWS SSM.
// Accepts the parameter type.
func PutParameter(client *ssm.Client, name, value string, paramType ParameterType) error {
	// Prepare the input for the PutParameter API call.
	input := &ssm.PutParameterInput{
		Name:      aws.String(name),               // Parameter name/path.
		Value:     aws.String(value),              // Parameter value.
		Type:      types.ParameterType(paramType), // Use the specified type (e.g., "String", "SecureString").
		Overwrite: aws.Bool(true),                 // Allow overwriting existing parameters.
	}

	// Call the SSM API to put the parameter.
	_, err := client.PutParameter(context.TODO(), input)
	return err
}

// GenerateTaskDefFromEnv reads a .env file and generates a task definition JSON with secrets.
func GenerateTaskDefFromEnv(envFile, outputFile, prefix string) error {
	// Read the .env file.
	data, err := os.ReadFile(envFile)
	if err != nil {
		return fmt.Errorf("failed to read env file: %w", err)
	}

	// Parse the .env content, handling multiline for certs.
	lines := strings.Split(string(data), "\n")
	var secrets []ExtendedSecret
	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			i++
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			i++
			continue // Skip invalid lines
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// Accumulate multiline values until the next key=value line
		for j := i + 1; j < len(lines); j++ {
			nextLine := strings.TrimSpace(lines[j])
			if matched, _ := regexp.MatchString(`^[A-Z_][A-Z0-9_]*=`, nextLine); matched {
				// Next line looks like a new key=value, stop accumulating
				i = j - 1 // Set i to j-1 so i++ will process the next key
				break
			} else if nextLine != "" { // Skip empty lines but accumulate non-empty
				value += "\n" + nextLine
			}
			if j == len(lines)-1 {
				i = j // If end of file, set i to last
			}
		}
		// Detect if it's a secret based on key name and value.
		paramType := detectParameterType(key, value)
		secret := ExtendedSecret{
			Name:      key,
			ValueFrom: prefix + key,
			Type:      paramType,
			Value:     value,
		}
		secrets = append(secrets, secret)
		i++
	}

	// Create the task definition.
	taskDef := TaskDefinition{
		ContainerDefinitions: []ContainerDefinition{
			{
				Secrets: secrets,
			},
		},
	}

	// Marshal to JSON.
	jsonData, err := json.MarshalIndent(taskDef, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write to output file.
	err = os.WriteFile(outputFile, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", outputFile, err)
	}

	fmt.Printf("Generated task definition saved to %s\n", outputFile)
	return nil
}

// detectParameterType determines if a parameter is a secret based on the key name and value patterns.
func detectParameterType(key, value string) ParameterType {
	lowerKey := strings.ToLower(key)
	// Check key for secret keywords
	secretKeywords := []string{"password", "secret", "key", "token", "api", "auth", "credential", "private", "cert", "ssl", "secure"}
	for _, keyword := range secretKeywords {
		if strings.Contains(lowerKey, keyword) {
			return SecureStringType
		}
	}

	// Check value for secret patterns using regex
	// Certificate: starts with -----BEGIN
	if strings.Contains(value, "-----BEGIN") {
		return SecureStringType
	}
	// Potential API key: long alphanumeric with some symbols
	if matched, _ := regexp.MatchString(`^[A-Za-z0-9+/=]{20,}$`, value); matched {
		return SecureStringType
	}
	// URL with credentials: http://user:pass@...
	if matched, _ := regexp.MatchString(`^https?://[^@]+@`, value); matched {
		return SecureStringType
	}
	// JWT-like: three parts separated by dots
	if matched, _ := regexp.MatchString(`^[A-Za-z0-9+/=]+\.[A-Za-z0-9+/=]+\.[A-Za-z0-9+/=]+$`, value); matched {
		return SecureStringType
	}
	// Base64-like long string
	if len(value) > 20 {
		if matched, _ := regexp.MatchString(`^[A-Za-z0-9+/=\-\n]+$`, value); matched {
			return SecureStringType
		}
	}

	return StringType
}
