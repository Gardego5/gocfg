package env

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/Gardego5/gocfg"
	"github.com/Gardego5/gocfg/utils"
)

// EnvLoader loads configuration from environment variables
func New() gocfg.Loader { return &loader{} }

type loader struct{}

func (*loader) GocfgLoaderName() string { return "env" }

func (e *loader) Load(
	ctx context.Context,
	field reflect.StructField, value reflect.Value,
	resolvedTag string,
) error {
	// Parse the tag to determine what to load
	tag := strings.TrimSpace(resolvedTag)

	// Handle special case - fully resolved reference or concatenation
	if strings.HasPrefix(tag, "@") || strings.Contains(tag, "||") {
		// At this point the tag should be resolved already
		return fmt.Errorf("unexpected unresolved tag: %s", tag)
	}

	// Parse the tag format (VAR, VAR?, VAR=default)
	var envVar string
	var defaultValue string
	var isOptional bool

	if strings.HasSuffix(tag, "?") {
		isOptional = true
		envVar = strings.TrimSuffix(tag, "?")
	} else if idx := strings.Index(tag, "="); idx >= 0 {
		envVar = strings.TrimSpace(tag[:idx])
		defaultValue = tag[idx+1:] // Preserve spaces in default value
	} else {
		envVar = tag
	}

	// Look up the environment variable
	envValue, exists := os.LookupEnv(envVar)
	if !exists {
		if isOptional {
			return nil // Optional field, no error if not set
		}
		if defaultValue != "" {
			// Use default value
			return utils.SetFieldValue(value, defaultValue)
		}
		return fmt.Errorf("%w: environment variable %s not set", utils.ErrMissingRequired, envVar)
	}

	return utils.SetFieldValue(value, envValue)
}
