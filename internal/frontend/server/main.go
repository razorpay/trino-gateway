package server

import (
	"context"
	"net/http"

	"github.com/NYTimes/gziphandler"
)

func NewServerHandler(ctx *context.Context) *http.Handler {
	guiFs := http.FileServer(http.Dir("./web/frontend"))
	appFrontendPath := "/"
	h := cacheHandler(
		compressionHandler(
			http.StripPrefix(appFrontendPath, guiFs),
		),
	)

	return &h
}

func cacheHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=180")
		h.ServeHTTP(w, r)
	})
}

func compressionHandler(h http.Handler) http.Handler {
	// TODO: check https://github.com/CAFxX/httpcompression
	return gziphandler.GzipHandler(h)
}
