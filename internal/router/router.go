package router

import (
	"net/http"

	"timetable-editor/internal/handler"
	"timetable-editor/web"
)

// New maps app URLs to their handlers and adds common response headers.
func New(app *handler.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", app.Home)
	mux.HandleFunc("POST /analyze", app.AnalyzeCalendar)
	mux.HandleFunc("POST /create", app.CreateFeed)
	mux.HandleFunc("POST /filter-groups/{id}/toggle", app.ToggleEventGroup)
	mux.HandleFunc("POST /filter-groups/{id}/state", app.UpdateEventGroupState)
	mux.HandleFunc("POST /filter-groups/{id}/test-state", app.ChangeTestState)
	mux.HandleFunc("POST /filter-groups/{id}/apply-test-state", app.ApplyTestState)
	mux.HandleFunc("POST /filter-groups/{id}/confirm-toggle", app.ConfirmEventGroupToggle)
	mux.HandleFunc("POST /filter-groups/{id}/cancel", app.CancelTestConfirmation)
	mux.HandleFunc("GET /feed/{token}/calendar.ics", app.Calendar)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(web.StaticFiles()))))
	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
		next.ServeHTTP(w, r)
	})
}
