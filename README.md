# Go Parameter Store Tool

A simple Golang CLI tool to get and put parameters in AWS Systems Manager Parameter Store, retrieve all parameters from an ECS task definition JSON file, push parameters from a custom template, and generate task definitions from .env files.

## Prerequisites

- Go 1.21 or later
- AWS credentials configured (e.g., via AWS CLI or environment variables)
- Appropriate IAM permissions for SSM Parameter Store

## Installation

Clone the repository and use the Makefile for easy installation:

```bash
git clone https://github.com/jahrulnr/parameter-store-tools.git
cd salter-aws
make install
```

This builds the tool and installs it to your Go bin directory (e.g., `~/go/bin/salter-aws`). Ensure `~/go/bin` is in your PATH. Bash completion is also installed to `~/.bash_completion.d/`.

To update to the latest version:
```bash
make update
```

To uninstall:
```bash
make uninstall
```

## Configuration

Create a `config.json` file in the project directory to customize settings:

```json
{
  "parameterPrefix": "/preprod/testing/",
  "region": "ap-southeast-3"
}
```

- `parameterPrefix`: Default prefix for parameter paths (used in generate action).
- `region`: Default AWS region if not specified via `-region` flag.

If `config.json` is missing, defaults are used.

## Usage

Run the tool from the project directory (all commands support `-region <aws-region>`, defaults to config or `ap-southeast-3`):

- **Get a single parameter**:
  ```bash
  salter-aws -action get -name /my/param
  ```
  Use `salter-aws -action get -h` for detailed help.

- **Get all parameters under a prefix**:
  ```bash
  salter-aws -action get-by-prefix -prefix /my/prefix/ -o output
  ```
  Retrieves all parameters starting with the prefix and saves them as `key=value` pairs in `output.env` (keys are stripped of the prefix) and as a task-definition JSON in `output.json`.
  Use `salter-aws -action get-by-prefix -h` for detailed help.

- **Put (set) a single parameter** (with optional type):
  ```bash
  salter-aws -action put -name /my/param -value "new value" -type string
  ```
  Supported types: `string`, `stringlist`, `securestring` (defaults to `string`).
  Use `salter-aws -action put -h` for detailed help.

- **Get all parameters from an ECS task definition JSON file** (print to console):
  ```bash
  salter-aws -s template/task-definition.json
  ```
  Parses the `secrets` array and outputs in `NAME=value` format.

- **Get all parameters and save to dated .env file**:
  ```bash
  salter-aws -s template/task-definition.json -o env
  ```
  Saves as `env-ddmmyy.env` (e.g., `env-020126.env`) with parameters in `key=value` format.

- **Put parameters from a custom template JSON file**:
  ```bash
  salter-aws -action put-from-template -s template/task-definition-simple.json
  ```
  Pushes `secrets` from the template to SSM, using specified `type` and `value`.
  Use `salter-aws -action put-from-template -h` for detailed help.

  Example template JSON (`template/task-definition-simple.json`):
  ```json
  {
    "containerDefinitions": [
      {
        "secrets": [
          {
            "name": "DB_PASSWORD",
            "valueFrom": "/myapp/db/password",
            "type": "SecureString",
            "value": "supersecret123"
          },
          {
            "name": "API_KEY",
            "valueFrom": "/myapp/api/key",
            "type": "SecureString",
            "value": "abcdef123456"
          }
        ]
      }
    ]
  }
  ```
  See the [template](./template/task-definition-simple.json) folder for example files to understand how putter and getter operations work with ECS task definitions.

- **Generate task definition JSON from .env file**:
  ```bash
  salter-aws -action generate -s env-020126.env -o task-definition-generated.json
  ```
  Creates a task definition with `secrets` based on the .env file, using the `parameterPrefix` from `config.json`.
  Use `salter-aws -action generate -h` for detailed help.

## Building

To build a binary:
```bash
go build -o salter-aws main.go
```

Then run:
```bash
./salter-aws -action get -name /my/param
./salter-aws -action get-by-prefix -prefix /my/prefix/ -o output
```

## Notes

- Uses AWS SDK v2 for Go.
- Region defaults to `config.json` or `ap-southeast-3`; override with `-region <aws-region>`.
- For SecureString parameters, ensure KMS decrypt permissions if retrieving encrypted values.
- Template JSON should have `secrets` with `name`, `valueFrom`, `type`, and `value` for pusher functionality.
- Generated task definitions use the prefix from `config.json` for `valueFrom` paths.