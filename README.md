# BonBonn Game Tracker

Go service with an embedded HTML/JavaScript admin interface and MongoDB storage. The existing FACEIT endpoints retain their response strings, status codes and redirects. Streamer IDs and lobby URLs are now managed in MongoDB. The eight existing mappings are imported once; subsequent edits and deletions survive restarts.

## Run

```sh
cp .env.example .env
chmod 600 .env
# Set FACEIT_API_KEY, ADMIN_PASSWORD and both MongoDB passwords.
docker network inspect monitoring >/dev/null 2>&1 || docker network create monitoring
docker compose up -d --build --wait
```

The application listens at `127.0.0.1:12000`. Its final Docker image uses `scratch`, a stripped static Go binary, CA certificates and a non-root user. MongoDB 8.0 runs in a separate container, with the named volume `mongo_data`, authentication and no published database port. `deploy/mongo-init.js` creates an application user restricted to the configured database. Use URL-safe MongoDB credentials (for example random hexadecimal passwords), since Compose constructs the MongoDB URI from them.

MongoDB initialization credentials apply only to an empty volume. Changing `.env` does not change users in an existing database. Back up the volume before database upgrades; never use `docker compose down -v` on production data.

For native development, use Go 1.27.2 and configure `MONGO_URI` and `MONGO_DB_NAME` in `.env`, then run `go run ./cmd/server`. Existing variables `apiKey`, `mongoString`, `mongoDbName`, `mongoPlayerCollectionName` and `mongoMatchroomCollectionName` are also accepted.

## Admin

`https://admin.faceitlobby.com/` serves the admin interface. Login uses HTTP Basic Auth: username `ADMIN_USERNAME` (default `admin`) and password `ADMIN_PASSWORD` from the server's `.env`. The UI supports creating, editing, disabling and deleting streamer mappings, and viewing the latest link accesses with streamer ID, timestamp, match ID, redirect target and result. Recorded accesses are HTTP requests, including possible bot requests.

The Spieler tab lists every player recognized from a valid webhook Authorization UUID, including players without a configured streamer mapping. It shows their FACEIT username, unique match count and last webhook time. Clicking a username opens their complete stored match history with 50-row pagination. The Matches tab shows current live matches and the 20 most recent completed matches. Duplicate webhook deliveries do not increase game counts; closing events preserve previously captured map and team details. Existing matchroom documents are imported into this history once on startup. History covers received/stored matches, not all matches ever played on FACEIT.

The Webhooks tab lists recent POST requests received at `/match`, including their HTTP headers and raw request body. Up to the latest 500 requests can be shown; bodies larger than 100 KiB are truncated in this request log.

The configured lobby URL appears in `/getmatch`. `/redirect` retains the existing FACEIT matchroom destination. Disabling a mapping makes `/getmatch` use its default FACEIT link. Unknown valid player IDs continue to work with the public webhook and match endpoints.

Admin routes and assets require authentication. Requests on the public API host cannot access `/admin`; they receive 404. Browser mutations must originate from the same admin origin.

| Method | Admin endpoint | Body/result |
| --- | --- | --- |
| GET | `/admin/api/status` | Database status and active match count |
| GET | `/admin/api/streamers` | Array of `{id,name,url,enabled}` |
| POST | `/admin/api/streamers` | `{id,name,url,enabled}`; existing FACEIT player UUID required |
| PUT | `/admin/api/streamers/{id}` | `{name,url,enabled}` |
| DELETE | `/admin/api/streamers/{id}` | Delete mapping |
| GET | `/admin/api/events?limit=50` | Latest accesses, newest first |
| GET | `/admin/api/webhook-requests?limit=50` | Latest POST requests to `/match`, including headers and body |
| GET | `/admin/api/players` | Webhook players, username, unique match count, last webhook |
| GET | `/admin/api/players/{id}/matches?limit=50&offset=0` | `{matches,total,limit,offset}`; complete stored history |
| GET | `/admin/api/matches` | `{live,past}`; active games and latest 20 completed games |

## Public API

| Method | Endpoint | Parameters |
| --- | --- | --- |
| GET | `/elo`, `/checkelo` | `username` |
| GET | `/elobyid` | `id` = lowercase FACEIT UUID |
| GET | `/elobysteamid` | `id` = 17-digit Steam64 ID |
| GET | `/matchhistory` | `username`, `limit` = 4–100 |
| GET | `/getmatch`, `/redirect` | `key` = lowercase FACEIT UUID |
| POST | `/match` | Header `Authorization: <player UUID>`, FACEIT event body |
| POST | `/internal/api/getMultiplePlayers` | `{players:[{nickname,steam64ID,...}]}` |
| GET, POST | `/test` | `{"message":"Hello World!"}` |

Legacy details are intentional: ELO returns both `(csgo)` and optional `(cs2)` values with the original spacing; `/redirect` uses the original `/en/csgo/room/` URL, while the unconfigured `/getmatch` fallback uses `/en/cs2/room/`. `match_status_configuring` opens a match, duplicate active matches return 404, `match_object_created` returns `Invalid Request oder match is already closed`, and other events close the match with `Close match`. Incoming JSON and extended URL-encoded forms are supported.

`GET /healthz` checks MongoDB readiness. `/server health` provides the Docker health check. Structured JSON logs include request ID, route, status, duration and redirect details without logging credentials.

## Deployment

GitHub Actions runs `go vet` and race-enabled tests, including integration tests against MongoDB, then deploys the exact tested commit over SSH. Deployment runs on pushes to `main` and `refactor/go-mongodb-admin`, or via the workflow's manual branch selector. Secrets required in GitHub: `HOST`, `USERNAME`, `PASSWORD` (SSH private key), `PORT` and `SSH_HOST_FINGERPRINT`.

Server checkout and secret configuration: `/opt/faceit-game-tracker/.env` (mode 0600). The workflow preserves this file, builds the application, and runs `docker compose up -d --wait`. It idempotently appends the webhook site from `deploy/Caddyfile.faceitlobby` to `/etc/caddy/Caddyfile`, validates and reloads Caddy, then checks the webhook host over HTTPS. Caddy reaches the application as `bonbonn-api:12000` on the existing `monitoring` network.

Set A records for `api.faceitlobby.com`, `webhook.faceitlobby.com` and `admin.faceitlobby.com` to `84.247.175.201`; remove AAAA records pointing elsewhere. Caddy obtains HTTPS certificates automatically when DNS points to the server. Incoming FACEIT POST webhooks use `https://webhook.faceitlobby.com/match`. Existing streamer subdomains in the imported lobby mappings retain their existing URLs; their DNS must also reach the appropriate redirect service.

## Verification

```sh
go vet ./...
go test -race ./...
MONGO_TEST_URI='mongodb://test:password@127.0.0.1:27017/?authSource=admin' go test -race ./...
node --check internal/web/static/app.js
docker compose --env-file .env.example config --quiet
```

`scripts/faceit_fixture.py` serves deterministic FACEIT data on port 19083. `scripts/compatibility.py` compares two local instances with the same fixtures, checking response bodies, status codes, content types, CORS and redirects. Set `LEGACY_URL` and `GO_URL`. Use isolated test databases: the script opens and closes synthetic matches. The original TypeScript application is available in the Git history before this rewrite.
