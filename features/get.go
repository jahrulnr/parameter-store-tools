package features

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// getParametersFromFile reads an ECS task definition JSON file and retrieves all SSM parameters referenced in the secrets.
// It parses the JSON, extracts parameter ARNs, fetches values, and either prints them or saves to a dated JSON file.
func GetParametersFromFile(client *ssm.Client, filename, outputPrefix string) error {
	// Read the entire JSON file into memory.
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	// Unmarshal the JSON data into the TaskDefinition struct.
	var taskDef TaskDefinition
	err = json.Unmarshal(data, &taskDef)
	if err != nil {
		return err
	}

	// Ensure there is at least one container definition.
	if len(taskDef.ContainerDefinitions) == 0 {
		return fmt.Errorf("no container definitions found")
	}

	// Iterate over the secrets in the first container (assuming single container for simplicity).
	secrets := taskDef.ContainerDefinitions[0].Secrets
	envMap := make(map[string]string) // For saving to JSON if outputPrefix is provided.

	for _, secret := range secrets {
		// Extract the parameter name from the ARN.
		paramName := ExtractParameterName(secret.ValueFrom)
		if paramName == "" {
			fmt.Printf("Invalid ARN for %s: %s\n", secret.Name, secret.ValueFrom)
			continue
		}
		// Fetch the parameter value from SSM.
		val, err := GetParameter(client, paramName)
		if err != nil {
			fmt.Printf("Failed to get %s: %v\n", secret.Name, err)
			continue
		}
		if outputPrefix != "" {
			// Collect for JSON output.
			envMap[secret.Name] = val
		} else {
			// Print the result in environment variable format.
			fmt.Printf("%s=%s\n", secret.Name, val)
		}
	}

	// If outputPrefix is provided, save to .env file with date.
	if outputPrefix != "" {
		dateStr := time.Now().Format("020106") // ddmmyy format.
		outputFile := fmt.Sprintf("%s-%s.env", outputPrefix, dateStr)
		var content strings.Builder
		for key, value := range envMap {
			content.WriteString(fmt.Sprintf("%s=%s\n", key, value))
		}
		err = os.WriteFile(outputFile, []byte(content.String()), 0644)
		if err != nil {
			return fmt.Errorf("failed to write file %s: %w", outputFile, err)
		}
		fmt.Printf("Saved bulk env to %s\n", outputFile)
	}

	return nil
}

// getParameter retrieves a single parameter from AWS SSM, with decryption enabled for SecureStrings.
func GetParameter(client *ssm.Client, name string) (string, error) {
	// Prepare the input for the GetParameter API call.
	input := &ssm.GetParameterInput{
		Name:           aws.String(name),
		WithDecryption: aws.Bool(true), // Decrypt SecureString parameters.
	}

	// Call the SSM API to get the parameter.
	result, err := client.GetParameter(context.TODO(), input)
	if err != nil {
		return "", err
	}

	// Return the decrypted parameter value.
	return *result.Parameter.Value, nil
}

// getParametersByPrefix retrieves all parameters under a specified prefix from AWS SSM and saves them to a .env file and a task-definition JSON.
// Parameter names are stripped of the prefix for the key in .env, but full names used in JSON.
func GetParametersByPrefix(client *ssm.Client, prefix, outputBase string) error {
	// Build the content for the .env file and collect secrets for JSON.
	var envContent strings.Builder
	var secrets []ExtendedSecret

	// Paginate through all parameters under the prefix.
	var nextToken *string
	for {
		// Prepare the input for the GetParametersByPath API call.
		input := &ssm.GetParametersByPathInput{
			Path:           aws.String(prefix),
			Recursive:      aws.Bool(true),
			WithDecryption: aws.Bool(true), // Decrypt SecureString parameters.
			NextToken:      nextToken,
			MaxResults:     aws.Int32(10), // Max allowed is 10.
		}

		// Call the SSM API to get parameters by path.
		result, err := client.GetParametersByPath(context.TODO(), input)
		if err != nil {
			return err
		}

		// Process the parameters.
		for _, param := range result.Parameters {
			name := *param.Name
			// Strip the prefix from the parameter name to create the key for .env.
			key := strings.TrimPrefix(name, prefix)
			if key == name {
				// If prefix not found, use the full name (though unlikely).
				key = name
			}
			value := *param.Value
			envContent.WriteString(fmt.Sprintf("%s=%s\n", key, value))

			// Determine the parameter type.
			var paramType ParameterType
			switch param.Type {
			case "String":
				paramType = StringType
			case "StringList":
				paramType = StringListType
			case "SecureString":
				paramType = SecureStringType
			default:
				paramType = StringType
			}

			// Create secret for JSON.
			secret := ExtendedSecret{
				Name:      key,
				ValueFrom: name, // Full parameter name for valueFrom.
				Type:      paramType,
				Value:     value,
			}
			secrets = append(secrets, secret)
		}

		// Check if there are more pages.
		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	// Write the .env file.
	envFile := outputBase + ".env"
	err := os.WriteFile(envFile, []byte(envContent.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write .env file %s: %w", envFile, err)
	}

	// Create and write the task-definition JSON.
	taskDef := TaskDefinition{
		ContainerDefinitions: []ContainerDefinition{
			{
				Secrets: secrets,
			},
		},
	}
	jsonData, err := json.MarshalIndent(taskDef, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	jsonFile := outputBase + ".json"
	err = os.WriteFile(jsonFile, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write JSON file %s: %w", jsonFile, err)
	}

	fmt.Printf("Saved .env to %s and task-definition JSON to %s\n", envFile, jsonFile)
	return nil
}
