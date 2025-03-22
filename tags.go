package gocfg

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/Gardego5/gocfg/utils"
)

// parseTag parses a tag and extracts field references
func parseTag(tag string) (references []string, err error) {
	inEscape := false

	for i := 0; i < len(tag); i++ {
		c := tag[i]

		if inEscape {
			inEscape = false
			continue
		}

		if c == '"' {
			inEscape = true
			continue
		}

		if c == '@' && i+1 < len(tag) && isIdentChar(tag[i+1]) {
			// Extract field reference
			start := i + 1
			end := start
			for end < len(tag) && isIdentChar(tag[end]) {
				end++
			}

			fieldName := tag[start:end]
			references = append(references, fieldName)
			i = end - 1
		}
	}

	return references, nil
}

// isIdentChar returns true if c is a valid identifier character
func isIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_'
}

// resolveTag resolves all references in a tag using current field values
func resolveTag(tag string, configValue reflect.Value) (string, error) {
	// Handle concatenation with ||
	if strings.Contains(tag, "||") {
		parts := strings.Split(tag, "||")
		var resolvedParts []string

		for _, part := range parts {
			part = strings.TrimSpace(part)
			resolved, err := resolvePart(part, configValue)
			if err != nil {
				return "", err
			}
			resolvedParts = append(resolvedParts, resolved)
		}

		return strings.Join(resolvedParts, ""), nil
	}

	return resolvePart(tag, configValue)
}

// resolvePart resolves a single part of a tag (handling @Field and escape sequences)
func resolvePart(part string, configValue reflect.Value) (string, error) {
	if strings.HasPrefix(part, "@") {
		fieldName := part[1:]
		field := configValue.FieldByName(fieldName)
		if !field.IsValid() {
			return "", fmt.Errorf("%w: %s", utils.ErrUnboundVariable, fieldName)
		}
		return field.String(), nil
	}

	// Handle escape sequences
	var result strings.Builder
	inEscape := false

	for i := 0; i < len(part); i++ {
		c := part[i]

		if inEscape {
			result.WriteByte(c)
			inEscape = false
			continue
		}

		if c == '"' {
			inEscape = true
			continue
		}

		result.WriteByte(c)
	}

	return result.String(), nil
}
