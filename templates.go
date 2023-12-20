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

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

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

// New creates a new ViewRenderer configured via the functional options.
func New(opts ...Option) *ViewRenderer {
	r := &ViewRenderer{
		templates:     make(map[string]*View),
		templateFuncs: template.FuncMap{},
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
		return errors.Wrap(err, "failed to list using file pattern")
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
		return errors.Wrap(err, "failed to list using file pattern")
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
		return errors.Wrap(err, "failed to read file names using file pattern")
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
func (t *ViewRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	log.Ctx(c.Request().Context()).Debug().Str("name", name).Bool("autoReload", t.autoReload).Msg("Render")

	start := time.Now()

	tmpl, err := t.lookupTemplate(name)
	if err != nil {
		log.Ctx(c.Request().Context()).Error().Err(err).Str("name", name).Msg("failed to load template")

		return c.NoContent(http.StatusInternalServerError)
	}

	// use the name of the template, or layout if it exists
	execName := path.Base(tmpl.name)
	if tmpl.layout != "" {
		execName = path.Base(tmpl.layout)
	}

	err = tmpl.template.ExecuteTemplate(w, execName, data)
	if err != nil {
		log.Ctx(c.Request().Context()).Error().Err(err).Str("name", tmpl.name).Str("layout", tmpl.layout).Msg("render template failed")
		return err
	}

	log.Ctx(c.Request().Context()).Debug().Str("name", tmpl.name).Str("dur", time.Since(start).String()).Str("layout", tmpl.layout).Msg("execute template")

	return nil
}

// RenderToHTMLBlob renders a template document to a HTML blob.
func (t *ViewRenderer) RenderToHTMLBlob(c echo.Context, code int, name string, data any) error {
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
	layoutName := path.Base(tmpl.layout)

	log.Debug().Str("templateName", templateName).Str("layoutName", layoutName).Str("includes", tmpl.includes).Msg("register template")

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

	log.Debug().Strs("patterns", patterns).Msg("new template")

	tmpl.template, err = template.New(templateName).Funcs(t.templateFuncs).ParseFS(t.fsys, patterns...)
	if err != nil {
		return errors.Wrapf(err, "failed to parse template %s", tmpl.name)
	}

	t.templates[templateName] = tmpl

	return nil
}

func readFileNames(fsys fs.FS, patterns ...string) ([]string, error) {
	var filenames []string

	for _, pattern := range patterns {
		list, err := fs.Glob(fsys, pattern)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list using file pattern")
		}

		if len(list) == 0 {
			return nil, fmt.Errorf("template: pattern matches no files: %#q", pattern)
		}
		filenames = append(filenames, list...)
	}

	return filenames, nil
}
