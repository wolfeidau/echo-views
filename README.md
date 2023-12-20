# echo-views

This library provides a view renderer for the [echo](https://github.com/labstack/echo) web framework. I rewrote my existing template renderer so this is "inspired" by the https://github.com/wolfeidau/echo-go-templates.


[![GitHub Actions status](https://github.com/wolfeidau/echo-views/workflows/Go/badge.svg?branch=main)](https://github.com/wolfeidau/echo-views/actions?query=workflow%3AGo)
[![Go Report Card](https://goreportcard.com/badge/github.com/wolfeidau/echo-views)](https://goreportcard.com/report/github.com/wolfeidau/echo-views)
[![Documentation](https://godoc.org/github.com/wolfeidau/echo-views?status.svg)](https://godoc.org/github.com/wolfeidau/echo-views)

# Features

* Provides a renderer for echo.
* Provides a simple template engine that supports layouts, includes, functions and standalone fragments.
* Uses the go standard library's [html/template](https://pkg.go.dev/html/template) package.
* Supports reloading of templates each time they are rendered, which is useful for development.
* Internal logger interface to allow use of any logging library.
* Use an interface to decouple from echo 

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

# Testing



# License

This application is released under Apache 2.0 license and is copyright [Mark Wolfe](https://www.wolfe.id.au/?utm_source=echo-go-templates).