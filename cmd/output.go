package cmd

import (
	"encoding/json"
	"fmt"
	"io"
)

type outputFlag string

func (o *outputFlag) Set(s string) error {
	switch s {
	case "text", "json":
		*o = outputFlag(s)
		return nil
	default:
		return fmt.Errorf("must be \"text\" or \"json\"")
	}
}

func (o *outputFlag) String() string { return string(*o) }
func (o *outputFlag) Type() string   { return "format" }

type Result struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	URL     string `json:"url,omitempty"`
}

func isJSONOutput() bool {
	return outputFormat == "json"
}

func printJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
