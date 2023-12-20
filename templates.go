package templates

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"path"
	"time"
)

type Context interface {
	// Request returns `*http.Request`.
	Request() *http.Request
	// HTMLBlob sends an HTTP blob response with status code.
	HTMLBlob(code int, b []byte) error
	// NoContent sends a response with no body and a status code.
	NoContent(code int) error
}

// View stores the meta data for each view template, and whether it uses a layout.
type View struct {
	layout   string
	name     string
	includes string
	template *template.Template
}

// ViewRenderer contains the template renderer state.
type ViewRenderer struct {
	fsys          fs.FS
	autoReload    bool
	templates     map[string]*View
	templateFuncs template.FuncMap
	logger        Logger
}

// Option is a functional option for configuring the ViewRenderer.
type Option func(r *ViewRenderer)

// WithFS sets the filesystem to load templates from.
func WithFS(fsys fs.FS) Option {
	return func(r *ViewRenderer) {
		r.fsys = fsys
	}
}

// WithAutoReload enables or disables automatically reloading templates when they change.
func WithAutoReload(enabled bool) Option {
	return func(r *ViewRenderer) {
		r.autoReload = enabled
	}
}

// WithFuncs sets the template functions to use.
func WithFuncs(funcs template.FuncMap) Option {
	return func(r *ViewRenderer) {
		r.templateFuncs = funcs
	}
}

// WithLogger sets the logger to use.
func WithLogger(logger Logger) Option {
	return func(r *ViewRenderer) {
		r.logger = logger
	}
}

// New creates a new ViewRenderer configured via the functional options.
func New(opts ...Option) *ViewRenderer {
	r := &ViewRenderer{
		templates:     make(map[string]*View),
		templateFuncs: template.FuncMap{},
		logger:        &noopLogger{},
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// AddWithLayout register one or more templates using the provided layout.
func (t *ViewRenderer) AddWithLayout(layout string, patterns ...string) error {
	filenames, err := readFileNames(t.fsys, patterns...)
	if err != nil {
		return fmt.Errorf("failed to list using file pattern: %w", err)
	}

	for _, f := range filenames {
		tpl := &View{
			layout: layout,
			name:   f,
		}

		err = t.compileTemplate(tpl)
		if err != nil {
			return err
		}
	}

	return nil
}

// AddWithLayoutAndIncludes register one or more templates using the provided layout and includes.
func (t *ViewRenderer) AddWithLayoutAndIncludes(layout, includes string, patterns ...string) error {
	filenames, err := readFileNames(t.fsys, patterns...)
	if err != nil {
		return fmt.Errorf("failed to list using file pattern: %w", err)
	}

	for _, f := range filenames {
		tpl := &View{
			layout:   layout,
			includes: includes,
			name:     f,
		}

		err = t.compileTemplate(tpl)
		if err != nil {
			return err
		}
	}

	return nil
}

// Add add a template to the registry.
func (t *ViewRenderer) Add(patterns ...string) error {
	filenames, err := readFileNames(t.fsys, patterns...)
	if err != nil {
		return fmt.Errorf("failed to read file names using file pattern: %w", err)
	}

	for _, f := range filenames {
		tpl := &View{
			name: f,
		}

		err = t.compileTemplate(tpl)
		if err != nil {
			return err
		}
	}

	return nil
}

// Render renders a template document.
func (t *ViewRenderer) Render(w io.Writer, name string, data interface{}, c Context) error {
	t.logger.DebugCtx(c.Request().Context(), "Render", map[string]any{"name": name, "autoReload": t.autoReload})

	start := time.Now()

	tmpl, err := t.lookupTemplate(name)
	if err != nil {
		t.logger.ErrorCtx(c.Request().Context(), "failed to load template", err, map[string]any{"name": name})
		return c.NoContent(http.StatusInternalServerError)
	}

	// use the name of the template, or layout if it exists
	execName := path.Base(tmpl.name)
	if tmpl.layout != "" {
		execName = path.Base(tmpl.layout)
	}

	err = tmpl.template.ExecuteTemplate(w, execName, data)
	if err != nil {
		t.logger.ErrorCtx(c.Request().Context(), "failed to execute template", err, map[string]any{"name": tmpl.name, "layout": tmpl.layout})

		return err
	}

	t.logger.DebugCtx(c.Request().Context(), "Render complete", map[string]any{"name": name, "dur": time.Since(start).String(), "layout": tmpl.layout})

	return nil
}

// RenderToHTMLBlob renders a template document to a HTML blob.
func (t *ViewRenderer) RenderToHTMLBlob(c Context, code int, name string, data any) error {
	buf := new(bytes.Buffer)
	if err := t.Render(buf, name, data, c); err != nil {
		return err
	}
	return c.HTMLBlob(code, buf.Bytes())
}

func (t *ViewRenderer) lookupTemplate(name string) (*View, error) {
	tmpl, ok := t.templates[name]
	if !ok {
		return nil, fmt.Errorf("template not found: %s", name)
	}

	if !t.autoReload {
		return tmpl, nil
	}

	err := t.compileTemplate(tmpl)
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}

func (t *ViewRenderer) compileTemplate(tmpl *View) (err error) {
	templateName := path.Base(tmpl.name)

	t.logger.Debug("register template", map[string]any{"name": tmpl.name, "layout": tmpl.layout, "includes": tmpl.includes})
	//
	// the list of patterns varies depending on whether the template uses a layout or includes
	//
	patterns := make([]string, 0)

	// add the layout if it exists
	if tmpl.layout != "" {
		patterns = append(patterns, tmpl.layout)
	}

	// then add the includes if they exist
	if tmpl.includes != "" {
		patterns = append(patterns, tmpl.includes)
	}

	// finally add the template itself
	patterns = append(patterns, tmpl.name)

	tmpl.template, err = template.New(templateName).Funcs(t.templateFuncs).ParseFS(t.fsys, patterns...)
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", tmpl.name, err)
	}

	t.templates[templateName] = tmpl

	return nil
}

func readFileNames(fsys fs.FS, patterns ...string) ([]string, error) {
	var filenames []string

	for _, pattern := range patterns {
		list, err := fs.Glob(fsys, pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to list using file pattern: %w", err)
		}

		if len(list) == 0 {
			return nil, fmt.Errorf("template: pattern matches no files: %#q", pattern)
		}
		filenames = append(filenames, list...)
	}

	return filenames, nil
}
