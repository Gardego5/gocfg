// Utilities for implementing your own loader for configurations.
package utils

import (
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
)

// SetFieldValue sets the appropriate value on the field based on its type
func SetFieldValue(fieldValue reflect.Value, value string) error {

	switch val := fieldValue.Addr().Interface().(type) {

	case encoding.TextUnmarshaler:
		return val.UnmarshalText([]byte(value))

	case encoding.BinaryUnmarshaler:
		return val.UnmarshalBinary([]byte(value))

	case json.Unmarshaler:
		return val.UnmarshalJSON([]byte(value))

	default:
		switch fieldValue.Kind() {

		case reflect.String:
			fieldValue.SetString(value)

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			intValue, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid integer value: %s", value)
			}
			fieldValue.SetInt(intValue)

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			uintValue, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid unsigned integer value: %s", value)
			}
			fieldValue.SetUint(uintValue)

		case reflect.Bool:
			boolValue, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("invalid boolean value: %s", value)
			}
			fieldValue.SetBool(boolValue)

		case reflect.Float32:
			floatValue, err := strconv.ParseFloat(value, 32)
			if err != nil {
				return fmt.Errorf("invalid float value: %s", value)
			}
			fieldValue.SetFloat(floatValue)

		case reflect.Float64:
			floatValue, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("invalid float value: %s", value)
			}
			fieldValue.SetFloat(floatValue)

		default:
			return fmt.Errorf("unsupported field type: %s", fieldValue.Kind())

		}
	}

	return nil
}
