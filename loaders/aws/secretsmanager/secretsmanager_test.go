package secretsmanager_test

import (
	"context"
	"encoding/json"
	"testing"

	. "github.com/Gardego5/gocfg"
	. "github.com/Gardego5/gocfg/loaders/aws/secretsmanager"
	"github.com/Gardego5/gocfg/loaders/env"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockSecretsManagerClient implements a mock for AWS SecretsManager client
type MockSecretsManagerClient struct {
	Secrets map[string]string // Map of secret name to secret value
}

// GetSecretValue implements the SecretsManager GetSecretValue operation
func (m *MockSecretsManagerClient) GetSecretValue(
	ctx context.Context,
	params *secretsmanager.GetSecretValueInput,
	optFns ...func(*secretsmanager.Options),
) (*secretsmanager.GetSecretValueOutput, error) {
	secretName := aws.ToString(params.SecretId)

	secretValue, exists := m.Secrets[secretName]
	if !exists {
		return nil, &types.ResourceNotFoundException{
			Message: aws.String("Secret " + secretName + " not found"),
		}
	}

	return &secretsmanager.GetSecretValueOutput{
		SecretString: aws.String(secretValue),
	}, nil
}

func setupMockClient() *MockSecretsManagerClient {
	// Initialize mock client with predefined secrets
	mockClient := &MockSecretsManagerClient{
		Secrets: map[string]string{
			"string-secret":   "simple-secret-value",
			"json-secret":     `{"username": "admin", "password": "secret123", "port": 5432, "enabled": true, "tags": ["prod", "secure"], "nested": {"key": "value"}}`,
			"app/database":    `{"url": "postgres://user:pass@localhost:5432/db", "Username": "dbuser", "Password": "dbpass"}`,
			"testapp/secrets": `{"apiKey": "test-api-key"}`,
		},
	}
	return mockClient
}

func TestSecretsManagerLoader(t *testing.T) {
	loader := New(setupMockClient())

	ctx := context.Background()

	t.Run("Loads simple string secret", func(t *testing.T) {
		const expected = "simple-secret-value"

		result, err := Load[struct {
			Value string `aws/secretsmanager:"string-secret"`
		}](ctx, loader)

		require.NoError(t, err)
		assert.Equal(t, expected, result.Value)
	})

	t.Run("Loads specific key from JSON secret", func(t *testing.T) {
		const expectedUsername = "admin"
		const expectedPassword = "secret123"

		result, err := Load[struct {
			Username string `aws/secretsmanager:"json-secret:username"`
			Password string `aws/secretsmanager:"json-secret:password"`
		}](ctx, loader)

		require.NoError(t, err)
		assert.Equal(t, expectedUsername, result.Username)
		assert.Equal(t, expectedPassword, result.Password)
	})

	t.Run("Loads numeric and boolean values from JSON", func(t *testing.T) {
		result, err := Load[struct {
			Port    int  `aws/secretsmanager:"json-secret:port"`
			Enabled bool `aws/secretsmanager:"json-secret:enabled"`
		}](ctx, loader)

		require.NoError(t, err)
		assert.Equal(t, 5432, result.Port)
		assert.True(t, result.Enabled)
	})

	t.Run("Handles optional secrets", func(t *testing.T) {
		result, err := Load[struct {
			Value string `aws/secretsmanager:"nonexistent-secret?"`
		}](ctx, loader)

		require.NoError(t, err)
		assert.Equal(t, "", result.Value)
	})

	t.Run("Handles optional keys in secrets", func(t *testing.T) {
		result, err := Load[struct {
			Value string `aws/secretsmanager:"json-secret:nonexistent-key?"`
		}](ctx, loader)

		require.NoError(t, err)
		assert.Equal(t, "", result.Value)
	})

	t.Run("Errors on missing required secrets", func(t *testing.T) {
		_, err := Load[struct {
			Value string `aws/secretsmanager:"nonexistent-secret"`
		}](ctx, loader)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "nonexistent-secret")
	})

	t.Run("Errors on missing required keys", func(t *testing.T) {
		_, err := Load[struct {
			Value string `aws/secretsmanager:"json-secret:nonexistent-key"`
		}](ctx, loader)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "nonexistent-key")
	})

	t.Run("Uses field name as key when no key specified", func(t *testing.T) {
		const expected = "dbuser"

		result, err := Load[struct {
			Username string `aws/secretsmanager:"app/database"`
		}](ctx, loader)

		require.NoError(t, err)
		assert.Equal(t, expected, result.Username)
	})

	t.Run("Handles references to other fields", func(t *testing.T) {
		const expectedAppName = "testapp"
		const expectedApiKey = "test-api-key"

		t.Setenv("APP_NAME", expectedAppName)

		result, err := Load[struct {
			AppName string `env:"APP_NAME"`
			ApiKey  string `aws/secretsmanager:"@AppName||/secrets:apiKey"`
		}](ctx, env.New(), loader)

		require.NoError(t, err)
		assert.Equal(t, expectedAppName, result.AppName)
		assert.Equal(t, expectedApiKey, result.ApiKey)
	})

	t.Run("Reports circular dependencies", func(t *testing.T) {
		_, err := Load[struct {
			A string `aws/secretsmanager:"@B"`
			B string `aws/secretsmanager:"@A"`
		}](ctx, loader)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "circular dependency")
	})

	t.Run("Loads complex types as JSON strings", func(t *testing.T) {
		result, err := Load[struct {
			Tags   string `aws/secretsmanager:"json-secret:tags"`
			Nested string `aws/secretsmanager:"json-secret:nested"`
		}](ctx, loader)

		require.NoError(t, err)

		// Verify the tags array was serialized correctly
		var tags []string
		err = json.Unmarshal([]byte(result.Tags), &tags)
		require.NoError(t, err)
		assert.Equal(t, []string{"prod", "secure"}, tags)

		// Verify the nested object was serialized correctly
		var nested map[string]string
		err = json.Unmarshal([]byte(result.Nested), &nested)
		require.NoError(t, err)
		assert.Equal(t, "value", nested["key"])
	})

	t.Run("Loads multiple sources in correct order", func(t *testing.T) {
		t.Setenv("DB_PORT", "5432")

		result, err := Load[struct {
			// From environment
			Port int `env:"DB_PORT"`

			// From secrets
			DBUser string `aws/secretsmanager:"app/database:Username"`
			DBPass string `aws/secretsmanager:"app/database:Password"`

			// Constructed connection string using both sources
			DBUrl string `env:"postgres://@DBUser:@DBPass\"@localhost:@Port/mydb"`
		}](ctx, env.New(), loader)

		require.NoError(t, err)
		assert.Equal(t, 5432, result.Port)
		assert.Equal(t, "dbuser", result.DBUser)
		assert.Equal(t, "dbpass", result.DBPass)
		assert.Equal(t, "postgres://dbuser:dbpass@localhost:5432/mydb", result.DBUrl)
	})
}

// Integration test with real AWS (commented out, uncomment for real testing)
/*
func TestWithRealAWS(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Load AWS configuration from environment or ~/.aws
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-west-2"))
	require.NoError(t, err)

	// Create a real Secrets Manager client
	smClient := secretsmanager.NewFromConfig(awsCfg)
	loader := SecretsManagerLoader{Client: smClient}

	// Test with real secrets - ensure these exist in your AWS account
	result, err := Load[struct {
		DBConnection string `aws/secretsmanager:"my-app/database:connection"`
	}](ctx, loader)

	require.NoError(t, err)
	assert.NotEmpty(t, result.DBConnection)
}
*/
