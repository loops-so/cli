package cmd

import (
	"runtime/debug"
	"strings"

	"github.com/loops-so/loops-go"
	"github.com/spf13/cobra"
)

var (
	version    = "dev"
	commit     = "none"
	sdkVersion = ""
)

func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		if version == "dev" {
			if info.Main.Version != "" && info.Main.Version != "(devel)" {
				version = info.Main.Version
			}
			for _, s := range info.Settings {
				if s.Key == "vcs.revision" && len(s.Value) >= 7 {
					commit = s.Value[:7]
					break
				}
			}
		}
		for _, dep := range info.Deps {
			if dep.Path == "github.com/loops-so/loops-go" {
				sdkVersion = dep.Version
				break
			}
		}
	}
	parts := []string{}
	if commit != "" && commit != "none" {
		parts = append(parts, "git "+commit)
	}
	if sdkVersion != "" {
		parts = append(parts, "sdk "+sdkVersion)
	}
	parts = append(parts, "spec "+loops.SpecVersion)
	suffix := ""
	if len(parts) > 0 {
		suffix = " (" + strings.Join(parts, ", ") + ")"
	}
	cobra.AddTemplateFunc("versionArt", versionArt)
	rootCmd.SetVersionTemplate("{{versionArt}}\n{{with .Name}}{{printf \"%s \" .}}{{end}}{{printf \"version %s\" .Version}}" + suffix + "\n")
}
