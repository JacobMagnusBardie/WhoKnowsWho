# Testing

## What we test

| Type | Where | What it checks |
|---|---|---|
| Unit | `backend/queries_test.go` | `escapeLikePattern` and `searchPages`: matching on title and content, language filter, `%` taken literally, description shortened |
| Integration | `backend/integration_test.go`, `backend/main_test.go` | Register → login → logout through the real router (`newMux`) on an `httptest` server with an in-memory SQLite database; register validation; `/api/search` without `q` |
| End-to-end | `e2e/tests/selectors.spec.js` | A real browser (Playwright, Chromium) searches, registers, logs in and logs out by clicking the selectors the simulator uses |
| API contract | `make spec-check` | The generated OpenAPI spec in `backend/docs/` matches the code |

All of them run in CI (`.github/workflows/ci.yaml`) on every pull request to `dev` and `main`.

## How to run

```bash
# Go unit and integration tests
cd backend
go test ./...

# End-to-end (starts the app itself on port 8080)
cd e2e
npm ci
npx playwright install chromium
npx playwright test
```

## Test types we skipped, and why

- **More browsers (Firefox, Edge, etc):** the end-to-end test checks that our selectors exist and work. That doesn't depend on the browser, and each extra browser makes CI slower.
- **Load and performance tests:** simulator performance is not graded, and the simulator itself already puts realistic load on the production app.
- **Smoke test after deploy:** `deploy.yaml` already checks that the app answers before it tags a release. A fuller smoke test is tracked in #80.
- **Weather page:** it calls an external API (Open-Meteo). Testing it would need a fake server for that API, which isn't worth it for a page outside the simulator's scope.
- **Fuzz and mutation testing:** useful for parsers and critical logic, but our inputs are already handled by SQL placeholders and `escapeLikePattern`, which has a table test.
- **Visual regression (screenshot comparison):** the frontend may change freely; only the selectors must stay, and the end-to-end test covers those.