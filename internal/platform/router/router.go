package router

import (
	"net/http"
)

// Middleware wraps an http.Handler with extra behavior.
type Middleware func(http.Handler) http.Handler

// Router wraps http.ServeMux to support global middleware chaining
// while still using Go 1.22's built-in method+path pattern matching.
type Router struct {
	mux         *http.ServeMux
	middlewares []Middleware
}

func New() *Router {
	return &Router{
		mux: http.NewServeMux(),
	}
}

// Use registers a global middleware — applied to every route in order added.
func (r *Router) Use(mw Middleware) {
	r.middlewares = append(r.middlewares, mw)
}

// Handle registers a route. pattern follows Go 1.22 syntax: "GET /events/{id}"
func (r *Router) Handle(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc(pattern, handler)
}

// Chain applies all registered middlewares to the final handler (mux),
// in the order they were added — first added = outermost layer.
func (r *Router) Chain() http.Handler {
	var handler http.Handler = r.mux
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		handler = r.middlewares[i](handler)
	}
	return handler
}