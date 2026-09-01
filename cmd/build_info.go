package cmd

import (
	"runtime/debug"
	"strings"

	"github.com/loops-so/loops-go"
)

var (
	version    = "dev"
	commit     = "none"
	sdkVersion = ""
)

// includes a leading newline to make it easy to read the ascii art here in the source
const versionHeader = `
    __    ____  ____  ____  _____
   / /   / __ \/ __ \/ __ \/ ___/
  / /   / / / / / / / /_/ /\__ \
 / /___/ /_/ / /_/ / ____/___/ /
/_____/\____/\____/_/    /____/
`

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
	header := strings.TrimPrefix(versionHeader, "\n")
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
	rootCmd.SetVersionTemplate(header + "\n{{with .Name}}{{printf \"%s \" .}}{{end}}{{printf \"version %s\" .Version}}" + suffix + "\n")
}
