package secretsmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/Gardego5/gocfg"
	"github.com/Gardego5/gocfg/utils"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type client interface {
	GetSecretValue(
		ctx context.Context,
		params *secretsmanager.GetSecretValueInput,
		optFns ...func(*secretsmanager.Options),
	) (*secretsmanager.GetSecretValueOutput, error)
}

// SecretsManagerLoader loads configuration from AWS Secrets Manager
func New(client client) gocfg.Loader { return &loader{client: client} }

type loader struct{ client client }

func (s *loader) GocfgLoaderName() string { return "aws/secretsmanager" }

// Load implements the Loader interface for AWS Secrets Manager
// Tag formats supported:
// - "secretName" - Get entire secret as JSON and use field name as key
// - "secretName:key" - Get specific key from a JSON secret
// - "secretName?" - Optional secret
// - "secretName:key?" - Optional key in secret
// - "@Field" - Reference another field for the secret name
// - "@Field||suffix" - Concatenate field value with a suffix
func (s *loader) Load(
	ctx context.Context,
	field reflect.StructField, value reflect.Value,
	resolvedTag string,
) error {
	// Handle special case - fully resolved reference or concatenation
	if strings.HasPrefix(resolvedTag, "@") || strings.Contains(resolvedTag, "||") {
		// At this point the tag should be resolved already
		return fmt.Errorf("unexpected unresolved tag: %s", resolvedTag)
	}

	// Parse the tag
	var secretName string
	var jsonKey string
	var isOptional bool

	// Check if secret is optional
	if strings.HasSuffix(resolvedTag, "?") {
		isOptional = true
		resolvedTag = strings.TrimSuffix(resolvedTag, "?")
	}

	// Check for JSON key specification
	if idx := strings.Index(resolvedTag, ":"); idx >= 0 {
		secretName = strings.TrimSpace(resolvedTag[:idx])
		jsonKey = strings.TrimSpace(resolvedTag[idx+1:])
	} else {
		secretName = strings.TrimSpace(resolvedTag)
		// If no key specified, use the field name as the key
		jsonKey = field.Name
	}

	// Get the secret value from AWS Secrets Manager
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}

	result, err := s.client.GetSecretValue(ctx, input)
	if err != nil {
		// Check if the error is because the secret doesn't exist
		if isOptional {
			return nil // Skip this field if it's optional
		}
		return fmt.Errorf("failed to retrieve secret %s: %w", secretName, err)
	}

	var secretValue string
	if result.SecretString != nil {
		secretValue = *result.SecretString
	} else if result.SecretBinary != nil {
		// Handle binary secrets if needed
		return fmt.Errorf("binary secrets not supported for field %s", field.Name)
	} else {
		return fmt.Errorf("empty secret returned for %s", secretName)
	}

	// Try to parse the secret as JSON
	var secretMap map[string]interface{}
	if err := json.Unmarshal([]byte(secretValue), &secretMap); err != nil {
		// Not a JSON object, use the whole string
		if jsonKey != "" && jsonKey != field.Name {
			return fmt.Errorf("cannot extract key %s from non-JSON secret %s", jsonKey, secretName)
		}
		return utils.SetFieldValue(value, secretValue)
	}

	// Extract the specific key from the JSON
	if jsonValue, exists := secretMap[jsonKey]; exists {
		// Convert the value to string based on its type
		var stringValue string
		switch v := jsonValue.(type) {
		case string:
			stringValue = v
		case float64:
			if v == float64(int(v)) {
				stringValue = fmt.Sprintf("%.0f", v)
			} else {
				stringValue = fmt.Sprintf("%g", v)
			}
		case bool:
			stringValue = fmt.Sprintf("%t", v)
		case nil:
			stringValue = ""
		default:
			// For complex types, re-encode as JSON
			bytes, err := json.Marshal(v)
			if err != nil {
				return fmt.Errorf("failed to marshal complex secret value: %w", err)
			}
			stringValue = string(bytes)
		}

		return utils.SetFieldValue(value, stringValue)
	}

	if isOptional {
		return nil
	}

	return fmt.Errorf("key %s not found in secret %s", jsonKey, secretName)
}
