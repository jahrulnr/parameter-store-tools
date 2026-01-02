// Package main is the entry point for the AWS Parameter Store CLI tool.
// It provides functionality to get and put individual parameters, or retrieve all parameters from an ECS task definition JSON file.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"go-param-store/features"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// main is the entry point. It parses command-line flags and executes the appropriate action.
func main() {
	// Define command-line flags for different operations.
	action := flag.String("action", "", "Action to perform: 'get', 'put', 'put-from-template', 'generate', or 'get-by-prefix'")
	name := flag.String("name", "", "Parameter name")
	value := flag.String("value", "", "Parameter value (required for 'put')")
	sourceFile := flag.String("s", "", "Source file (JSON for get/put-from-template, .env for generate)")
	paramType := flag.String("type", "string", "Parameter type: 'string', 'stringlist', or 'securestring' (defaults to 'string')")
	outputPrefix := flag.String("o", "", "Output prefix for saving bulk env (e.g., 'env' saves as 'env-ddmmyy.env') or output file for generate/get-by-prefix")
	region := flag.String("region", "", "AWS region (defaults to config or 'ap-southeast-3')")
	prefix := flag.String("prefix", "", "Prefix for get-by-prefix action")
	helpFlag := flag.Bool("h", false, "Show help for the specified action")
	flag.Parse()

	// Show help if requested
	if *helpFlag {
		showHelp(*action)
		return
	}

	// Load configuration from config.json if exists.
	toolConfig, err := features.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	// Set default region from config if not specified.
	if *region == "" {
		*region = toolConfig.Region
	}
	// Handle generate action (no AWS needed).
	if *action == "generate" {
		if *sourceFile == "" || *outputPrefix == "" {
			fmt.Println("Error: -s <env-file> and -o <output.json> required for 'generate'")
			os.Exit(1)
		}
		err := features.GenerateTaskDefFromEnv(*sourceFile, *outputPrefix, toolConfig.ParameterPrefix)
		if err != nil {
			log.Fatalf("Failed to generate task definition: %v", err)
		}
		return
	}

	// Load AWS configuration with the specified region for SSM operations.
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(*region))
	if err != nil {
		log.Fatalf("Unable to load SDK config: %v", err)
	}

	// Create an SSM client using the loaded configuration.
	client := ssm.NewFromConfig(cfg)

	// Handle put-from-template action.
	if *action == "put-from-template" {
		if *sourceFile == "" {
			fmt.Println("Error: -s <filename.json> is required for 'put-from-template'")
			os.Exit(1)
		}
		err := features.PutParametersFromTemplate(client, *sourceFile)
		if err != nil {
			log.Fatalf("Failed to put parameters from template: %v", err)
		}
		return
	}

	// If a source file is provided, retrieve parameters from the ECS task definition.
	if *sourceFile != "" {
		err := features.GetParametersFromFile(client, *sourceFile, *outputPrefix)
		if err != nil {
			log.Fatalf("Failed to get parameters from file: %v", err)
		}
		return
	}

	// Validate required flags based on action.
	if *action == "" {
		fmt.Println("Usage:")
		fmt.Println("  Individual parameter operations:")
		fmt.Println("    go run main.go -action <get|put> -name <param-name> [-value <param-value>] [-type <type>] [-region <region>]")
		fmt.Println("")
		fmt.Println("  Bulk operations from ECS task definition:")
		fmt.Println("    go run main.go -s <filename.json> [-o <output-prefix>] [-region <region>]")
		fmt.Println("")
		fmt.Println("  Generate task definition from .env:")
		fmt.Println("    go run main.go -action generate -s <env-file> -o <output.json>")
		fmt.Println("")
		fmt.Println("  Get parameters by prefix:")
		fmt.Println("    go run main.go -action get-by-prefix -prefix <prefix> -o <output-base>")
		fmt.Println("")
		fmt.Println("  Put from template:")
		fmt.Println("    go run main.go -action put-from-template -s <template.json>")
		os.Exit(1)
	}
	if (*action == "get" || *action == "put") && *name == "" {
		fmt.Println("Error: -name is required for 'get' and 'put' actions")
		os.Exit(1)
	}
	if *action == "put" && *value == "" {
		fmt.Println("Error: -value is required for 'put' action")
		os.Exit(1)
	}
	if *action == "generate" && (*sourceFile == "" || *outputPrefix == "") {
		fmt.Println("Error: -s <env-file> and -o <output.json> required for 'generate'")
		os.Exit(1)
	}
	if *action == "put-from-template" && *sourceFile == "" {
		fmt.Println("Error: -s <filename.json> is required for 'put-from-template'")
		os.Exit(1)
	}
	if *action == "get-by-prefix" && (*prefix == "" || *outputPrefix == "") {
		fmt.Println("Error: -prefix and -o <output-base> required for 'get-by-prefix'")
		os.Exit(1)
	}

	// Execute the specified action.
	switch *action {
	case "get":
		// Retrieve a single parameter.
		val, err := features.GetParameter(client, *name)
		if err != nil {
			log.Fatalf("Failed to get parameter: %v", err)
		}
		fmt.Printf("Parameter %s: %s\n", *name, val)
	case "get-by-prefix":
		// Retrieve all parameters under a prefix.
		if *prefix == "" || *outputPrefix == "" {
			fmt.Println("Error: -prefix and -o <output-base> required for 'get-by-prefix'")
			os.Exit(1)
		}
		err := features.GetParametersByPrefix(client, *prefix, *outputPrefix)
		if err != nil {
			log.Fatalf("Failed to get parameters by prefix: %v", err)
		}
	case "put":
		// Ensure value is provided for put operation.
		if *value == "" {
			fmt.Println("Error: -value is required for 'put' action")
			os.Exit(1)
		}
		// Validate type.
		validTypes := map[string]bool{"string": true, "stringlist": true, "securestring": true}
		if !validTypes[strings.ToLower(*paramType)] {
			fmt.Println("Error: Invalid type. Use 'string', 'stringlist', or 'securestring'")
			os.Exit(1)
		}
		// Capitalize type for API.
		apiType := strings.ToLower(*paramType)
		switch apiType {
		case "string":
			apiType = "String"
		case "stringlist":
			apiType = "StringList"
		case "securestring":
			apiType = "SecureString"
		}
		// Store a parameter with the specified type.
		err := features.PutParameter(client, *name, *value, features.ParameterType(apiType))
		if err != nil {
			log.Fatalf("Failed to put parameter: %v", err)
		}
		fmt.Printf("Parameter %s set successfully as %s\n", *name, *paramType)
	default:
		// Handle invalid actions.
		fmt.Println("Invalid action. Use 'get', 'put', 'put-from-template', 'generate', or 'get-by-prefix'")
		os.Exit(1)
	}
}

func showHelp(action string) {
	switch action {
	case "get":
		fmt.Println("Help for 'get' action:")
		fmt.Println("  Retrieve a single parameter from AWS SSM.")
		fmt.Println("  Usage: salter-aws -action get -name <param-name> [-region <region>]")
		fmt.Println("  Example: salter-aws -action get -name /my/param")
	case "put":
		fmt.Println("Help for 'put' action:")
		fmt.Println("  Store or update a single parameter in AWS SSM.")
		fmt.Println("  Usage: salter-aws -action put -name <param-name> -value <value> [-type <type>] [-region <region>]")
		fmt.Println("  Types: string, stringlist, securestring (default: string)")
		fmt.Println("  Example: salter-aws -action put -name /my/param -value 'hello' -type securestring")
	case "put-from-template":
		fmt.Println("Help for 'put-from-template' action:")
		fmt.Println("  Push parameters from a JSON template to AWS SSM.")
		fmt.Println("  Usage: salter-aws -action put-from-template -s <template.json> [-region <region>]")
		fmt.Println("  Template format: ECS task definition with 'secrets' array.")
		fmt.Println("  Example: salter-aws -action put-from-template -s template/task-definition.json")
	case "generate":
		fmt.Println("Help for 'generate' action:")
		fmt.Println("  Generate an ECS task definition JSON from a .env file.")
		fmt.Println("  Usage: salter-aws -action generate -s <env-file> -o <output.json>")
		fmt.Println("  Automatically detects parameter types (string, securestring, etc.).")
		fmt.Println("  Example: salter-aws -action generate -s my.env -o task-def.json")
	case "get-by-prefix":
		fmt.Println("Help for 'get-by-prefix' action:")
		fmt.Println("  Retrieve all parameters under a prefix from AWS SSM.")
		fmt.Println("  Usage: salter-aws -action get-by-prefix -prefix <prefix> -o <output-base> [-region <region>]")
		fmt.Println("  Saves to <output-base>.env and <output-base>.json")
		fmt.Println("  Example: salter-aws -action get-by-prefix -prefix /prod/app/ -o app-params")
	default:
		fmt.Println("General help:")
		fmt.Println("  Use -action <action> -h for specific help.")
		fmt.Println("  Actions: get, put, put-from-template, generate, get-by-prefix")
		fmt.Println("  Example: salter-aws -action get -h")
	}
}
