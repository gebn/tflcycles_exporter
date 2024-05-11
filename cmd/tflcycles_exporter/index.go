package main

import (
	"bytes"
	_ "embed"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/gebn/go-stamp/v2"
)

var (
	//go:embed index.html
	indexTmpl string
)

func renderIndex() ([]byte, error) {
	tmpl := template.Must(template.New("index").Parse(indexTmpl))
	// This could be a strings.Builder, however template.Template.Execute()
	// takes an io.Writer, and keeping the underlying bytes as-is saves a
	// conversion from string to []byte.
	buf := bytes.Buffer{}
	data := struct {
		Stamp string
	}{
		stamp.Summary(),
	}
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// buildIndexHandler returns an http.Handler implementation that writes the
// landing page for the exporter. This page is efficient to produce, so can be
// used for health checking the process.
func buildIndexHandler(logger *slog.Logger) (http.Handler, error) {
	response, err := renderIndex()
	if err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if _, err := w.Write(response); err != nil {
			logger.ErrorContext(ctx, "failed to write response",
				slog.String("error", err.Error()))
		}
	}), nil
}
