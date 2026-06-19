package main

import (
	"flag"
	"runtime/debug"
)

func registerVersionFlags(fs *flag.FlagSet, printVersion *bool) {
	fs.BoolVar(printVersion, "version", false, "Print version and exit")
	fs.BoolVar(printVersion, "v", false, "Print version and exit")
}

func resolvedVersion() string {
	if version != "" && version != "dev" {
		return version
	}

	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		moduleVersion := buildInfo.Main.Version
		if moduleVersion != "" && moduleVersion != "(devel)" {
			return moduleVersion
		}
	}

	return "dev"
}
