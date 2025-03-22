// gocfg provides a declarative way to load configuration for you go application
// from a variety of sources.
package gocfg

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type Loader interface {
	// Load loads values into the field based on the tag
	Load(ctx context.Context, field reflect.StructField, value reflect.Value, resolvedTag string) error

	// Name returns the name of the loader (used for tag lookup)
	GocfgLoaderName() string
}

// Load loads configuration into a struct of type C using the provided loaders
func Load[C any](ctx context.Context, loaders ...Loader) (config C, err error) {
	config = *new(C)

	// Get type information for the config struct
	configValue := reflect.ValueOf(&config).Elem()
	configType := configValue.Type()

	// Map loaders by name for easy lookup
	loaderMap := make(map[string]Loader)
	for _, loader := range loaders {
		loaderMap[loader.GocfgLoaderName()] = loader
	}

	// Build dependency graph
	nodes := make(map[string]*node)

	// First pass: discover all fields and their dependencies
	for i := 0; i < configType.NumField(); i++ {
		field := configType.Field(i)

		for loaderName, loader := range loaderMap {
			tag := field.Tag.Get(loaderName)
			if tag == "" {
				continue
			}

			// Clean whitespace from the tag
			tag = strings.TrimSpace(tag)

			// Parse dependencies from the tag
			deps, err := parseTag(tag)
			if err != nil {
				return config, fmt.Errorf("error parsing tag for %s: %w", field.Name, err)
			}

			nodes[field.Name] = &node{
				fieldName:    field.Name,
				fieldIndex:   i,
				tag:          tag,
				loader:       loader,
				dependencies: deps,
			}

			break // Only use the first loader that matches
		}
	}

	// Check for circular dependencies
	if err := detectCircularDependencies(nodes); err != nil {
		return config, err
	}

	// Process nodes in dependency order
	for len(nodes) > 0 {
		progress := false

		for fieldName, n := range nodes {
			// Check if all dependencies are resolved
			allResolved := true
			for _, dep := range n.dependencies {
				if node, exists := nodes[dep]; exists && !node.resolved {
					allResolved = false
					break
				}
			}

			if allResolved {
				progress = true

				field := configType.Field(n.fieldIndex)
				fieldValue := configValue.Field(n.fieldIndex)

				// Resolve references in the tag
				resolvedTag, err := resolveTag(n.tag, configValue)
				if err != nil {
					return config, fmt.Errorf("error resolving tag for %s: %w", fieldName, err)
				}

				// Load the value using the appropriate loader
				if err := n.loader.Load(ctx, field, fieldValue, resolvedTag); err != nil {
					return config, fmt.Errorf("error loading %s: %w", fieldName, err)
				}

				// Mark as resolved and remove from pending nodes
				n.resolved = true
				delete(nodes, fieldName)
			}
		}

		if !progress {
			return config, errors.New("unable to resolve all dependencies, possible circular reference")
		}
	}

	return config, nil
}
