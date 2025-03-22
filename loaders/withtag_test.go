package loaders_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Gardego5/gocfg"
	"github.com/Gardego5/gocfg/loaders"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLoader implements the gocfg.Loader interface for testing
type MockLoader struct {
	NameToReturn string
	LoadFunc     func(ctx context.Context, field reflect.StructField, value reflect.Value, resolvedTag string) error
	LoadCalls    []LoadCall
}

type LoadCall struct {
	Field       reflect.StructField
	Value       reflect.Value
	ResolvedTag string
}

func (m *MockLoader) GocfgLoaderName() string {
	return m.NameToReturn
}

func (m *MockLoader) Load(ctx context.Context, field reflect.StructField, value reflect.Value, resolvedTag string) error {
	m.LoadCalls = append(m.LoadCalls, LoadCall{
		Field:       field,
		Value:       value,
		ResolvedTag: resolvedTag,
	})

	if m.LoadFunc != nil {
		return m.LoadFunc(ctx, field, value, resolvedTag)
	}
	return nil
}

// StringLoader is a simple loader that sets string values
type StringLoader struct {
	NameToReturn string
	PrefixToAdd  string
}

func (s *StringLoader) GocfgLoaderName() string {
	return s.NameToReturn
}

func (s *StringLoader) Load(_ context.Context, _ reflect.StructField, value reflect.Value, resolvedTag string) error {
	if value.Kind() != reflect.String {
		return errors.New("field must be a string")
	}
	value.SetString(s.PrefixToAdd + resolvedTag)
	return nil
}

func TestWithTag(t *testing.T) {
	t.Run("Returns the specified tag name", func(t *testing.T) {
		underlyingLoader := &MockLoader{NameToReturn: "original"}
		wrappedLoader := loaders.WithTag("custom", underlyingLoader)

		assert.Equal(t, "custom", wrappedLoader.GocfgLoaderName())
	})

	t.Run("Delegates Load calls to the underlying loader", func(t *testing.T) {
		// Create a mock loader that tracks calls
		mockLoader := &MockLoader{NameToReturn: "original"}
		wrappedLoader := loaders.WithTag("custom", mockLoader)

		// Create a struct with a field to load
		type TestConfig struct {
			Value string `custom:"test-value"`
		}

		config := TestConfig{}
		field, _ := reflect.TypeOf(config).FieldByName("Value")
		value := reflect.ValueOf(&config).Elem().FieldByName("Value")

		// Call Load on the wrapped loader
		err := wrappedLoader.Load(context.Background(), field, value, "test-value")
		require.NoError(t, err)

		// Verify the call was delegated to the mock loader
		require.Len(t, mockLoader.LoadCalls, 1)
		assert.Equal(t, "test-value", mockLoader.LoadCalls[0].ResolvedTag)
		assert.Equal(t, field, mockLoader.LoadCalls[0].Field)
		// Can't directly compare reflect.Value, so check its string representation
		assert.Equal(t, value.String(), mockLoader.LoadCalls[0].Value.String())
	})

	t.Run("Propagates errors from the underlying loader", func(t *testing.T) {
		expectedErr := errors.New("test error")
		mockLoader := &MockLoader{
			LoadFunc: func(ctx context.Context, field reflect.StructField, value reflect.Value, resolvedTag string) error {
				return expectedErr
			},
		}
		wrappedLoader := loaders.WithTag("custom", mockLoader)

		// Call Load on the wrapped loader
		err := wrappedLoader.Load(context.Background(), reflect.StructField{}, reflect.Value{}, "anything")
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("Works with different loader types", func(t *testing.T) {
		// Use the StringLoader which adds a prefix to string values
		stringLoader := &StringLoader{
			NameToReturn: "string",
			PrefixToAdd:  "PREFIX-",
		}

		// Wrap it with a custom tag
		wrappedLoader := loaders.WithTag("custom-string", stringLoader)

		// Load a configuration using the wrapped loader
		type TestConfig struct {
			Value string `custom-string:"test-value"`
		}

		ctx := context.Background()
		result, err := gocfg.Load[TestConfig](ctx, wrappedLoader)
		require.NoError(t, err)

		// Check that the StringLoader's prefix was added
		assert.Equal(t, "PREFIX-test-value", result.Value)
	})

	t.Run("Integration with multiple loaders", func(t *testing.T) {
		// Create two string loaders with different prefixes
		loader1 := &StringLoader{
			NameToReturn: "original1",
			PrefixToAdd:  "ONE-",
		}
		loader2 := &StringLoader{
			NameToReturn: "original2",
			PrefixToAdd:  "TWO-",
		}

		// Wrap them with custom tags
		wrappedLoader1 := loaders.WithTag("custom1", loader1)
		wrappedLoader2 := loaders.WithTag("custom2", loader2)

		// Define a config that uses both custom tags
		type TestConfig struct {
			Value1 string `custom1:"test1"`
			Value2 string `custom2:"test2"`
		}

		// Load the config with both wrapped loaders
		ctx := context.Background()
		result, err := gocfg.Load[TestConfig](ctx, wrappedLoader1, wrappedLoader2)
		require.NoError(t, err)

		// Check that each loader processed its own tag
		assert.Equal(t, "ONE-test1", result.Value1)
		assert.Equal(t, "TWO-test2", result.Value2)
	})
}
