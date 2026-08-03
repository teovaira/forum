package webutil

import "net/http"

// RegisterRoutes mounts the /static/* file server on mux, serving files
// straight from dir (e.g. "web/static") for CSS, JS, and any other public
// assets. It follows the same RegisterRoutes(mux, ...) shape every other
// package uses to wire its routes, so cmd/server can register all of them
// the same way.
//
// Parameters:
//   - mux: The ServeMux to register the /static/* route on.
//   - dir: The directory on disk to serve static files from.
func RegisterRoutes(mux *http.ServeMux, dir string) {
	fileServer := http.FileServer(http.Dir(dir))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))
}
