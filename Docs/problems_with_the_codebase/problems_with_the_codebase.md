# Problems with the legacy codebase

Mandatory I. Findings from reviewing the inherited WhoKnows Flask application
(`src/`), sorted by priority — most critical first.

Scope: backend (`src/backend/app.py`), frontend (templates and CSS), database
schema, and the build/run tooling. Line references are to the legacy code as of
tag `v0.1.0`.

---

## Critical

### 1. SQL injection in every database query

Every query builds SQL with Python string formatting instead of parameter binding:

| Location | Function |
|---|---|
| `app.py:63` | `get_user_id` |
| `app.py:79` | `before_request` |
| `app.py:97` | `search` |
| `app.py:133` | `api_search` |
| `app.py:143` | `api_login` |
| `app.py:169` | `api_register` |

`/api/search?q=` is unauthenticated and reachable by anyone, and `api_login`
means authentication can be bypassed outright. Ranked first because it is
trivially exploitable, needs no credentials, and exposes the whole database.

**Direction:** parameterised queries everywhere. In the rewrite, prefer a driver
or query layer that makes string interpolation awkward or impossible.

### 2. Passwords hashed with unsalted MD5

`app.py:186-190`. MD5 is broken for password storage: it is fast by design,
which is the opposite of what password hashing needs, and without a salt
identical passwords produce identical hashes. The seeded admin hash
(`5f4dcc3b5aa765d61d8327deb882cf99`) is the MD5 of `password` and can be
reversed by a lookup table in seconds.

**Direction:** a slow, salted algorithm — bcrypt or argon2.

### 3. Hardcoded session secret

`app.py:19` — `SECRET_KEY = 'development key'`, committed to a public
repository. Flask signs session cookies with this key, so anyone who reads the
source can forge a session and log in as any user. This is a third independent
route to full account takeover, alongside findings 1 and 2.

**Direction:** load from the environment, never commit it, rotate the value that
has been public.

### 4. Database committed to version control

`src/whoknows.db` was tracked in git, containing real user rows and password
hashes. Removing it from tracking does not remove it from history.

**Direction:** ignore generated databases; decide deliberately whether the
history needs rewriting. Note this file also held the entire `pages` search
corpus (~1 MB), so it is data we still need somewhere outside git.

---

## High

### 5. Flask development server used as the production server

`app.py:200` binds `0.0.0.0:8080` using Flask's built-in server, which is
single-threaded and explicitly not intended for production use.

**Direction:** a real application server behind a reverse proxy.

### 6. Dependencies frozen at 2010 releases

`src/backend/requirements.txt` pins Flask 0.5, Werkzeug 0.6.1, Jinja2 2.4 and
setuptools 44.1.1. Years of unpatched vulnerabilities, and none of it installs
on a current Python. Because Werkzeug and Jinja2 are pinned *directly* rather
than resolved through Flask, upgrading Flask alone is not possible — see the
dependency graph.

### 7. Search does a full table scan

`app.py:97` and `app.py:133` use `content LIKE '%<query>%'` against the `pages`
table with no index and no full-text search. Cost grows linearly with the
corpus on every keystroke-driven request. This is the component most likely to
fall over first under simulated traffic.

**Direction:** full-text search (SQLite FTS5 or equivalent).

### 8. A request can terminate the server

`check_db_exists` calls `sys.exit(1)` at `app.py:41`. It runs via `connect_db()`
inside `before_request`, so a missing database file kills the whole process
while serving a request rather than returning an error.

### 9. No error handling on form input

`app.py:143` and `app.py:169` read `request.form['username']` directly. Any POST
missing a field raises `KeyError` and returns a 500. All user input is trusted
to exist and to be well-formed.

### 10. Search results render a column that does not exist

`search.html:14` outputs `{{ result.description }}`, but the `pages` table has
no `description` column — it has `content`. Every result renders an empty
description. A silent template failure: Jinja2 resolves the missing attribute to
nothing rather than raising.

---

## Medium

### 11. The "API" is not an API

`/api/login` and `/api/register` return rendered HTML and redirects rather than
data, and no endpoint sets a meaningful status code. Page routes and API routes
are mixed in one module with no separation. This blocks generating a
specification from the running server.

### 12. No configuration separation

All configuration lives in module-level constants at `app.py:16-19`. Nothing
reads the environment — no `os.environ` or `getenv` anywhere in `src/`. Settings
cannot differ between development and production without editing source.

`DATABASE_PATH` is currently the absolute `/tmp/whoknows.db`, which does not
exist on Windows, so the project cannot be run by part of the team.

### 13. The test suite does not test the application

`app_tests.py:12` sets `app.DATABASE` to a temporary file, but the application
reads `DATABASE_PATH`. The override silently does nothing, so all 84 lines of
tests run against the real database — while `schema.sql:1` begins with
`DROP TABLE IF EXISTS users`. There is no CI to run them in any case.

### 14. `run_forever.sh` restart loop is unreliable

The original reported the wrong exit code (`$?` was overwritten by the `if` test
before use), left `$PYTHON_SCRIPT_PATH` unquoted, and carried an unused
variable. A clean exit also re-loops with no delay. Partially addressed in #4;
process supervision belongs in the operating system rather than a shell loop.

### 15. No logging or monitoring

Nothing records requests, errors, or timings. There is no way to observe the
application in operation, and no signal when something fails.

---

## Low

### 16. Invalid and inaccessible HTML

`layout.html` has no `<html>`, `<head>`, `<body>`, no `<meta charset>` — despite
the page title containing `¿` — no viewport meta tag, and no `lang` attribute
even though the application supports English and Danish. Several elements are
left unclosed, relying on browser error recovery.

No `<label>` elements exist in any template, so no form input has an accessible
name. Forms use `<dl>/<dt>/<dd>` for layout, which is a description list rather
than a form structure, and presentational attributes such as `size="30"`.

### 17. Search does not work without JavaScript

`search.html` puts the search box outside any `<form>`, with no `name`
attribute, wired to an inline `onclick` handler. If the script fails to load,
the core feature of the site is unusable.

### 18. Dated stylesheet

`style.css:28-29,38-39` still carries `-moz-` and `-webkit-border-radius`
prefixes that have been unnecessary for over a decade. Layout is done with
floats rather than flexbox or grid, and the page is not responsive. The footer
reads `© 2009`.

### 19. Repository hygiene

Vim swap files (`.app.py.swn`, `.app.py.swo`) and `app.py.bak` were committed
alongside the source.

---

## How this was reviewed

Manual reading of `src/`, cross-checked against the schema and templates.
`shellcheck` was used on `run_forever.sh`. Prioritisation weighs exploitability
without credentials first, then data loss, then availability under the simulated
traffic, then correctness, then maintainability.

Findings 1, 2, 3, 6, 7 and 12 are the ones the rewrite should be designed to
prevent rather than fix afterwards.
