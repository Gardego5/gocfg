package gocfg_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	. "github.com/Gardego5/gocfg"
	"github.com/Gardego5/gocfg/loaders/env"
)

func TestLoadEnv(t *testing.T) {
	t.Run("Returns an error given no environment variable set", func(t *testing.T) {
		if _, err := Load[struct {
			Value string `env:"VALUE"`
		}](context.Background(), env.New()); err == nil {
			t.Fatal("expected error for missing VALUE")
		}
	})

	t.Run("Returns a struct with the environment variable when present", func(t *testing.T) {
		const expected = "test"
		t.Setenv("VALUE", expected)
		if env, err := Load[struct {
			Value string `env:"VALUE"`
		}](context.Background(), env.New()); err != nil {
			t.Fatal(err)
		} else if env.Value != expected {
			t.Fatalf("expected %s, got %s", expected, env.Value)
		}
	})

	t.Run("Handles optional values", func(t *testing.T) {
		if _, err := Load[struct {
			Value string `env:"VALUE?"`
		}](context.Background(), env.New()); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("Handles default values", func(t *testing.T) {
		const (
			expectedValue  = "default"
			expectedNumber = 42
		)

		if env, err := Load[struct {
			Value  string `env:"VALUE=default"`
			Number int    `env:"NUMBER=42"`
		}](context.Background(), env.New()); err != nil {
			t.Fatalf("unexpected error: %s", err)
		} else if env.Value != expectedValue {
			t.Fatalf("expected Value=%s, got %s", expectedValue, env.Value)
		} else if env.Number != expectedNumber {
			t.Fatalf("expected Number=%d, got %d", expectedNumber, env.Number)
		}
	})

	t.Run("Handles references to other fields", func(t *testing.T) {
		const (
			expectedVarName = "VALUE"
			expectedValue   = "test"
		)

		t.Setenv("VARNAME", expectedVarName)
		t.Setenv("VALUE", expectedValue)
		if env, err := Load[struct {
			VarName string `env:"VARNAME"`
			Value   string `env:"@VarName"`
		}](context.Background(), env.New()); err != nil {
			t.Fatalf("unexpected error: %s", err)
		} else if env.VarName != expectedVarName {
			t.Fatalf("expected VarName=%s, got %s", expectedVarName, env.VarName)
		} else if env.Value != expectedValue {
			t.Fatalf("expected Value=%s, got %s", expectedValue, env.Value)
		}
	})

	t.Run("Handles concatenation of values", func(t *testing.T) {
		const (
			expectedA      = "A"
			expectedB      = "B"
			expectedJoined = "CCC"
		)

		t.Setenv("AA", expectedA)
		t.Setenv("BB", expectedB)
		t.Setenv("AB", expectedJoined)
		if env, err := Load[struct {
			A      string `env:"AA"`
			B      string `env:"BB"`
			Joined string `env:"@A||@B"`
		}](context.Background(), env.New()); err != nil {
			t.Fatalf("unexpected error: %s", err)
		} else if env.A != expectedA {
			t.Fatalf("expected A=%s, got %s", expectedA, env.A)
		} else if env.B != expectedB {
			t.Fatalf("expected B=%s, got %s", expectedB, env.B)
		} else if env.Joined != expectedJoined {
			t.Fatalf("expected Joined=%s, got %s", expectedJoined, env.Joined)
		}
	})

	t.Run("Handles literals in concatenation", func(t *testing.T) {
		const expected = "test"

		t.Setenv("APP_CONFIG_VAR1", expected)
		if env, err := Load[struct {
			Prefix string `env:"PREFIX=APP_CONFIG_"`
			Var1   string `env:"@Prefix||VAR1"`
		}](context.Background(), env.New()); err != nil {
			t.Fatalf("unexpected error: %s", err)
		} else if env.Var1 != expected {
			t.Fatalf("expected Var1=%s, got %s", expected, env.Var1)
		}
	})

	t.Run("Cleans whitespace from tag values", func(t *testing.T) {
		const expectedKey1 = "value1"

		t.Setenv("KEY1", expectedKey1)
		if env, err := Load[struct {
			Key1 string `env:"  KEY1  "`       // strip whitespace from keys
			Key2 string `env:"KEY2  =  value"` // preserve whitespace after default value
		}](context.Background(), env.New()); err != nil {
			t.Fatalf("unexpected error: %s", err)
		} else if env.Key1 != expectedKey1 {
			t.Fatalf("expected Key1=%s, got %s", expectedKey1, env.Key1) // stripped
		} else if env.Key2 != "  value" {
			t.Fatalf("expected Key2='  value', got '%s'", env.Key2) // preserved
		}
	})

	t.Run("Reports circular dependencies", func(t *testing.T) {
		if _, err := Load[struct {
			A string `env:"@B"`
			B string `env:"@A"`
		}](context.Background(), env.New()); err == nil {
			t.Fatal("expected error for circular dependency")
		}

		if _, err := Load[struct {
			A string `env:"@B"`
			B string `env:"@C"`
			C string `env:"@A"`
		}](context.Background(), env.New()); err == nil {
			t.Fatal("expected error for circular dependency")
		}
	})

	t.Run("Detects unboud variables", func(t *testing.T) {
		if _, err := Load[struct {
			A string `env:"@B"`
		}](context.Background(), env.New()); err == nil {
			t.Fatal("expected error for unbound variable")
		}
	})

	t.Run("Allows escaping characters using '\"'", func(t *testing.T) {
		t.Skip("escaping test not implemented yet")

		const expected = "test"

		t.Setenv("@@SPECIAL", expected)
		if env, err := Load[struct {
			Special string `env:"\"@\"@SPECIAL"`
		}](context.Background(), env.New()); err != nil {
			t.Fatalf("unexpected error: %s", err)
		} else if env.Special != expected {
			t.Fatalf("expected Special=%s, got %s", expected, env.Special)
		}
	})
}

type jsonValue map[string]any

var _ json.Unmarshaler = (*jsonValue)(nil)

func (j *jsonValue) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*map[string]any)(j))
}

func TestLoadWithEncodings(t *testing.T) {
	t.Run("Unmarshals text values", func(t *testing.T) {
		t.Setenv("VALUE", "info")
		if env, err := Load[struct {
			LogLevel slog.Level `env:"VALUE"`
		}](context.Background(), env.New()); err != nil {
			t.Fatalf("unexpected error: %s", err)
		} else if env.LogLevel != slog.LevelInfo {
			t.Fatalf("expected Value=%s, got %s", slog.LevelInfo, env.LogLevel)
		}
	})

	t.Run("Unmarshals binary values", func(t *testing.T) {
		t.Skip("binary unmarshaling test implemented yet")
	})

	t.Run("Unmarshals JSON values", func(t *testing.T) {
		t.Setenv("VALUE", `{"key": "value"}`)
		if env, err := Load[struct {
			Value jsonValue `env:"VALUE"`
		}](context.Background(), env.New()); err != nil {
			t.Fatalf("unexpected error: %s", err)
		} else if len(env.Value) != 1 {
			t.Fatalf("expected Value to have 1 key, got %d", len(env.Value))
		} else if value, ok := env.Value["key"]; !ok {
			t.Fatalf("expected Value to have key 'key'")
		} else if value != "value" {
			t.Fatalf("expected Value['key']='value', got %s", value)
		}
	})
}
