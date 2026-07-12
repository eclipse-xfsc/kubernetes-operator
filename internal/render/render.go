package render

import "strings"

type Context struct {
	Namespace string
	Workload  string
	Type      string
	Provider  string
	Tenant    string
}

func Template(s string, ctx Context) string {
	repl := map[string]string{
		"{{ namespace }}":          ctx.Namespace,
		"{{ consumer.namespace }}": ctx.Namespace,
		"{{ workload }}":           ctx.Workload,
		"{{ consumer.name }}":      ctx.Workload,
		"{{ type }}":               ctx.Type,
		"{{ provider }}":           ctx.Provider,
		"{{ tenant }}":             ctx.Tenant,
	}
	out := s
	for k, v := range repl {
		out = strings.ReplaceAll(out, k, v)
	}
	return out
}
