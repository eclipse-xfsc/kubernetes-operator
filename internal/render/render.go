package render

import (
	"bytes"
	"text/template"
)

type Context struct {
	Consumer struct{ Name, Namespace, Kind string }
	Binding  struct{ Name, Namespace, Type string }
	Provider struct{ Name, Namespace, Type string }
}

func String(s string, ctx Context) (string, error) {
	t, err := template.New("xsfc").Option("missingkey=error").Parse(s)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := t.Execute(&b, ctx); err != nil {
		return "", err
	}
	return b.String(), nil
}

func Map(in map[string]string, ctx Context) (map[string]string, error) {
	out := map[string]string{}
	for k, v := range in {
		r, err := String(v, ctx)
		if err != nil {
			return nil, err
		}
		out[k] = r
	}
	return out, nil
}
