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

- **Load testing:** speed isn't graded, and loading the production VM would slow down the simulator's real users. We watch real traffic through monitoring instead.
- **More browsers:** our inputs and buttons are plain HTML that works the same in every browser, and each extra browser slows CI down.
- **Smoke test after deploy:** `deploy.yaml` already waits for the app to answer before it tags a release. A fuller check is tracked in #80.
- **Weather:** it calls an external API, so testing it needs a fake version of that API. Weather isn't part of the simulation.
- **Fuzz and mutation testing:** our input handling is small (`?` placeholders and `escapeLikePattern`), so there is little for them to find.
- **Visual regression:** the design may change freely. Only the selectors must stay, and the end-to-end test covers those.