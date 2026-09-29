# Architecture

The server renders the main form and uses HTMX to submit it. The handler reads the form, asks the backend to validate and save the feed, then returns a small HTML fragment for HTMX to place in the page. Calendar subscriptions follow a separate route: the backend looks up the saved configuration, fetches its iCal source, and returns the calendar unchanged. Optional filter settings are saved for later use.

```text
Browser --GET /, POST /create--> router --> handler --> backend --> database
Calendar app --GET /feed/{token}/calendar.ics--> router --> handler --> backend --> database
backend --GET source iCal--> University of Canterbury timetable
backend --unchanged iCal response--> Calendar app
```

## Folders

- `cmd/server` starts the HTTP server and opens the local page in a browser.
- `internal/router` maps request paths to handlers and applies response security headers.
- `internal/handler` translates HTTP forms and feed requests into backend calls.
- `internal/backend` validates source URLs, fetches iCal data, and encrypts saved configuration.
- `internal/database` owns the SQLite schema and repository methods.
- `web/templates` contains the full page and the small HTMX response fragments.
- `web/static` contains CSS and the pinned HTMX library.

## Stored data

SQLite stores an HMAC of each source URL and public token, plus an AES-256-GCM encrypted configuration. The encryption key is created once at first start in the operating system's per-user config folder; on Windows this is under the current user's AppData folder. It is separate from the SQLite file, so copying only the database does not reveal source URLs. Keep that key file if feeds must continue working. `TIMETABLE_MASTER_KEY` can override the local key for managed deployments.

The server binds to `127.0.0.1:8080` by default. It only fetches HTTPS iCal URLs on `timetable.canterbury.ac.nz` and rejects redirects to other hosts.
