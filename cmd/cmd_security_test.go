package cmd

import (
	"testing"

	"github.com/siliconflow/bizyair-cli/config"
)

func TestRedactedArgumentsDoesNotLeakOrMutateAPIKey(t *testing.T) {
	original := &config.Argument{ApiKey: "super-secret", Name: "model"}
	safe := redactedArguments(original)

	if safe.ApiKey != "[REDACTED]" {
		t.Fatalf("redacted API key = %q", safe.ApiKey)
	}
	if original.ApiKey != "super-secret" {
		t.Fatalf("original API key was mutated: %q", original.ApiKey)
	}
	if safe.Name != original.Name {
		t.Fatalf("non-sensitive argument was changed: %q", safe.Name)
	}
}

func TestRedactedArgumentsNil(t *testing.T) {
	if redactedArguments(nil) != nil {
		t.Fatal("nil arguments should remain nil")
	}
}
