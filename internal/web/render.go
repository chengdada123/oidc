package web

import (
	"html/template"
	"net/http"
	"path/filepath"
)

type Renderer struct {
	baseDir string
}

func NewRenderer(baseDir string) (*Renderer, error) {
	return &Renderer{baseDir: baseDir}, nil
}

func (r *Renderer) Render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t, err := template.ParseFiles(
		filepath.Join(r.baseDir, "internal", "web", "templates", "base.html"),
		filepath.Join(r.baseDir, "internal", "web", "templates", name),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := t.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
