# Metrics

The app counts a few things in memory and serves the current totals at `/metrics` in Prometheus format. Prometheus fetches them every few seconds and stores the history, which is what makes graphs over time possible.

## Where the endpoints are

- `GET /health` on the site's port (8080). Returns `200 {"status":"ok"}` when the app runs and the database answers, and `503` when the database doesn't. It is public, so Postman or an uptime check can use it.
- `GET /metrics` on its own port, `9091` (set with `METRICS_ADDR`). `compose.yaml` publishes 8080 to the internet for the simulator but does not publish 9091, so only containers on the same Docker network can reach it. That keeps the metrics private without depending on nginx, which port 8080 bypasses.

## What we measure and why

Each metric answers a question about how the site is used.

| Metric | Labels | Question it answers |
|---|---|---|
| `whoknows_http_requests_total` | `route`, `method`, `status` | Which pages and API routes are used, and how many requests fail (5xx)? |
| `whoknows_http_request_duration_seconds` | `route` | Is the site slow, and on which route? |
| `whoknows_searches_total` | `result`: `hits` / `empty` | How many searches are there, and how many find nothing? |
| `whoknows_registrations_total` | | Are new users signing up? |
| `whoknows_logins_total` | `result`: `ok` / `failed` | How often do logins fail? |

The client library also exports Go runtime and process metrics (`go_*`, `process_*`), such as memory use and CPU time of the app.

Empty searches are the one we expect to learn the most from. A high share means users look for content we don't have, and logging what they searched for (#128) tells us what to scrape (#129).

`route` is the matched route pattern such as `GET /api/search`, never the raw URL, so bots requesting random paths can't create an endless number of series. Requests that a middleware answers before routing (for example a user who must change their password) are labelled `unmatched`.

## Considered and left out

Session length (time from login to logout) would separate the simulator's bots from real users, but it needs the login time stored in the session cookie and only counts users who log out. We left it for later.
