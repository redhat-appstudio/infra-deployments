package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/ghodss/yaml"
	"github.com/microcosm-cc/bluemonday"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

// colorful output when users try it locally
var (
	infoLog    = color.New(color.FgCyan, color.Bold).PrintlnFunc()
	successLog = color.New(color.FgGreen, color.Bold).PrintlnFunc()
	errorLog   = color.New(color.FgRed, color.Bold).PrintlnFunc()
	normalLog  = color.New(color.Reset).PrintlnFunc()
)

// validateSchema validates the given JSON object against the provided schema.
func validateSchema(jsonObj interface{}, schema *jsonschema.Schema) error {
	if err := schema.Validate(jsonObj); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}
	return nil
}

// validateHTMLSafety ensures that HTML content in specific fields is safe.
func validateHTMLSafety(jsonObj interface{}) error {
	obj, ok := jsonObj.(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected a JSON object")
	}

	// sanitize details
	if details, ok := obj["details"].(string); ok && details != "" {
		cleaned := bluemonday.StrictPolicy().Sanitize(details)
		if cleaned != details {
			return fmt.Errorf("unsafe HTML content detected in 'details' field")
		}
	}

	// pure text
	for _, field := range []string{"title", "message"} {
		if value, ok := obj[field].(string); ok {
			if strings.Contains(value, "<") && strings.Contains(value, ">") {
				return fmt.Errorf("'%s' field must not contain HTML tags", field)
			}
		}
	}

	return nil
}

// validateContent performs both schema validation and HTML safety checks.
func validateContent(jsonObj interface{}, schema *jsonschema.Schema) error {
	if err := validateSchema(jsonObj, schema); err != nil {
		return err
	}
	if err := validateHTMLSafety(jsonObj); err != nil {
		return err
	}
	return nil
}

// readYamlFile reads and converts YAML file to a generic JSON-compatible object.
func readYamlFile(path string) (interface{}, error) {
	yamlBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	jsonBytes, err := yaml.YAMLToJSON(yamlBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to convert YAML to JSON: %w", err)
	}

	var obj interface{}
	if err := json.Unmarshal(jsonBytes, &obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return obj, nil
}

// loadSchema loads and compiles the JSON schema from the given path.
func loadSchema(path string) (*jsonschema.Schema, error) {
	schema, err := jsonschema.Compile("file://" + path)
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema: %w", err)
	}
	return schema, nil
}

// getBannerContentFilesInDir finds all banner-content.yaml or banner-content.yml files in the directory tree.
func getBannerContentFilesInDir(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == "banner-content.yaml" || d.Name() == "banner-content.yml" {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// validateFile performs full validation on a single YAML file.
func validateFile(path string, schema *jsonschema.Schema) error {
	infoLog("Validating:", path)

	jsonObj, err := readYamlFile(path)
	if err != nil {
		return fmt.Errorf("failed to load file for validation: %w", err)
	}

	if err := validateContent(jsonObj, schema); err != nil {
		return err
	}

	successLog("✅ Passed:", path)
	return nil
}

// main function to run the validator from command line.
func main() {
	if len(os.Args) < 3 {
		normalLog(fmt.Sprintf("Usage: %s <schema.json> <file.yaml>", os.Args[0]))
		os.Exit(1)
	}

	schemaPath := os.Args[1]

	// Load and compile schema
	schema, err := loadSchema(schemaPath)
	if err != nil {
		errorLog("Error loading schema:", err)
		os.Exit(1)
	}

	// Run validation
	folderPath := os.Args[2]
	filePaths, err := getBannerContentFilesInDir(folderPath)
	if err != nil {
		errorLog("Error collecting banner content YAML files: %v", err)
	}

	if len(filePaths) == 0 {
		normalLog("No banner-content.yaml files found for validation.")
		os.Exit(0)
	}

	hasError := false

	for _, filePath := range filePaths {
		if err := validateFile(filePath, schema); err != nil {
			errorLog("❌ Validation error in", filePath+":", err)
			hasError = true
		}
	}

	if hasError {
		errorLog("Validation completed with errors.")
		os.Exit(1)
	} else {
		successLog("🎉 All files passed validation successfully!")
	}
}
