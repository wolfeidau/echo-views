# echo-views

This library provides a view renderer for the [echo](https://github.com/labstack/echo) web framework. I rewrote my existing template renderer so this is "inspired" by the https://github.com/wolfeidau/echo-go-templates.

# Features

* Provides a renderer for echo.
* Provides a simple template engine that supports layouts, includes, functions and standalone fragments.
* Uses the go standard library's [html/template](https://pkg.go.dev/html/template) package.
* Supports reloading of templates each time they are rendered, which is useful for development.

# Usage

```go

	e := echo.New()

	viewRenderer := templates.New(
        templates.WithReload(true),
		templates.WithFS(os.DirFS("./views")),
		templates.WithFuncs(template.FuncMap{
			"getTime": func() string {
				return time.Now().Format("15:04:05")
			},
		}),
    )

    e.Renderer = viewRenderer
```

# License

This application is released under Apache 2.0 license and is copyright [Mark Wolfe](https://www.wolfe.id.au/?utm_source=echo-go-templates).