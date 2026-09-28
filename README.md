# Timetable Filter

A small Go server that creates filtered Google Calendar iCal feeds. It removes events when any phrase matches a selected title, description, or location field.

## Start locally

Requirements: Go 1.23 or newer and a C compiler for the SQLite driver.

```powershell
go run ./cmd/server
```

The server creates and reuses one encryption key in your per-user config folder, then opens `http://127.0.0.1:8080` in your browser. Leave the terminal open while using the site; press Ctrl+C to stop it.

Set `ADDR` to change the listen address, `TIMETABLE_DB_PATH` to move the SQLite file, `TIMETABLE_KEY_FILE` to choose another key path, and `OPEN_BROWSER=false` for a headless run. For a deployment behind HTTPS, set `PUBLIC_BASE_URL` to its origin.

The SQLite driver is the only Go module dependency. HTMX is a pinned static file served locally, with no custom browser JavaScript or frontend build step.

See [docs/architecture.md](docs/architecture.md) for the request flow and data storage design.
