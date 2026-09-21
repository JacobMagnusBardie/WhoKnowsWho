# ¿Who Knows?

Before Google, before Bing, before anyone thought to ask a chatbot — there was WhoKnows. A search engine from a simpler age, built in Flask, with the security posture of an unlocked bicycle. It went dormant. It should have stayed that way.

We have resurrected it in Go.

The living implementation is in [`backend/`](backend) and [`frontend/`](frontend). The original remains are preserved in [`src/`](src) for archaeological purposes only — new work goes in `backend/`/`frontend/`. A full autopsy of what we found (SQL injection, unsalted MD5, a hardcoded session secret, and other period-appropriate features) is in [`Docs/problems_with_the_codebase`](Docs/problems_with_the_codebase/problems_with_the_codebase.md). Do not deploy this to production. It has been dead once already.

## Prerequisites

- Go 1.26 or newer (see [`backend/go.mod`](backend/go.mod))

## Running

```bash
cd backend
go run .
```

Server starts on `http://localhost:8080`. The SQLite database (`backend/whoknows.db`) is created automatically on first run.

Test login: `testuser` / `password123`

## Routes

- Pages: `/`, `/login`, `/register`, `/weather`
- API: `/api/search`, `/api/login`, `/api/register`, `/api/logout`, `/api/weather`
- API docs: `/swagger/`

## Deployment

Pushing to `main` deploys automatically: [`deploy.yaml`](.github/workflows/deploy.yaml) builds the backend, copies the binary and `frontend/` to the Azure VM, and restarts the `whoknows` systemd service. The SQLite database lives on the VM and persists across deploys.

Pull requests and pushes to `dev`/`main` run [`ci.yaml`](.github/workflows/ci.yaml) (build, vet, test, lint).

## Regenerating the API spec

Endpoint docs are generated from `@Summary`/`@Router` annotations in [`backend/main.go`](backend/main.go) with [swag](https://github.com/swaggo/swag). The output in `backend/API-specs/` is committed, so regenerate and commit it after changing an endpoint's annotations:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
cd backend
swag init -g main.go -d . -o ./API-specs
```

## Conventions

Branching, commit messages, pull requests and releases: [`Docs/conventions_contract/`](Docs/conventions_contract)
