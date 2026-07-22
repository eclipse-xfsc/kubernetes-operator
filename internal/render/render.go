package render

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

type Context struct {
	Namespace string
	Workload  string
	Type      string
	Provider  string
	Tenant    string
}

// Render renders Go template expressions such as {{ .Workload }} and returns
// parsing or execution errors to the caller.
func Render(s string, ctx Context) (string, error) {
	legacy := map[string]string{
		"{{ namespace }}":          ctx.Namespace,
		"{{ consumer.namespace }}": ctx.Namespace,
		"{{ workload }}":           ctx.Workload,
		"{{ consumer.name }}":      ctx.Workload,
		"{{ type }}":               ctx.Type,
		"{{ provider }}":           ctx.Provider,
		"{{ tenant }}":             ctx.Tenant,
	}

	normalized := s
	for expression, value := range legacy {
		normalized = strings.ReplaceAll(normalized, expression, value)
	}

	tmpl, err := template.New("resource-output").Option("missingkey=error").Parse(normalized)
	if err != nil {
		return "", fmt.Errorf("parse resource output template: %w", err)
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, ctx); err != nil {
		return "", fmt.Errorf("execute resource output template: %w", err)
	}
	return rendered.String(), nil
}

// Template is the backwards-compatible convenience wrapper. Valid templates
// are rendered; invalid templates are returned unchanged.
func Template(s string, ctx Context) string {
	rendered, err := Render(s, ctx)
	if err != nil {
		return s
	}
	return rendered
}
