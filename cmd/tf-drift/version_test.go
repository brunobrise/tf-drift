package main

import (
	"flag"
	"io"
	"testing"
)

func TestRegisterVersionFlagsSupportsLongAndShortAliases(t *testing.T) {
	for _, args := range [][]string{{"-version"}, {"-v"}} {
		fs := flag.NewFlagSet("tf-drift", flag.ContinueOnError)
		fs.SetOutput(io.Discard)

		var printVersion bool
		registerVersionFlags(fs, &printVersion)

		if err := fs.Parse(args); err != nil {
			t.Fatalf("Parse(%v) failed: %v", args, err)
		}
		if !printVersion {
			t.Fatalf("expected %v to enable version output", args)
		}
	}
}

func TestResolvedVersionUsesInjectedVersion(t *testing.T) {
	original := version
	t.Cleanup(func() { version = original })

	version = "v1.2.3"

	if got := resolvedVersion(); got != "v1.2.3" {
		t.Fatalf("expected injected version, got %q", got)
	}
}

func TestResolvedVersionFallsBackToDevWhenBuildInfoHasNoModuleVersion(t *testing.T) {
	original := version
	t.Cleanup(func() { version = original })

	version = "dev"

	if got := resolvedVersion(); got == "" || got == "(devel)" {
		t.Fatalf("expected printable fallback version, got %q", got)
	}
}
