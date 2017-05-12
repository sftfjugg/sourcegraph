//go:generate go run generate.go

// Package bundle contains the bundled assets for the vscode
// application.
//
// To fetch the prebuilt vscode package, run:
//
//   cmd/frontend/internal/app/bundle/fetch-and-generate.bash
//
// To publish a vscode package, follow the steps in vscode-private/SOURCEGRAPH.md
package bundle

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/shurcooL/httpgzip"
)

var (
	noCache  bool
	errNoApp = errors.New("vscode app is not enabled on this server")
)

// Handler handles HTTP requests for files in the bundle.
func Handler() http.Handler {
	if Data == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, errNoApp.Error())
		})
	}

	fs := httpgzip.FileServer(Data, httpgzip.FileServerOptions{})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if noCache && os.Getenv("VSCODE_CACHE") == "" {
			w.Header().Set("Cache-Control", "no-cache")
		} else {
			w.Header().Set("Cache-Control", "max-age=300")
		}
		w.Header().Set("Vary", "Accept-Encoding")

		if name := path.Base(r.URL.Path); name == "index.html" || name == "webview.html" {
			// The UI uses iframes, so we need to allow iframes.
			w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		}

		fs.ServeHTTP(w, r)
	})
}
