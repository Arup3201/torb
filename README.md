# Torb (TaskOrbit)

Torb is a project-collaboration backend for organizing work between contributors. Users can discover projects, request to join them, work through assigned tasks, discuss tasks with other members, and follow project activity through stored notifications and a WebSocket channel.

This repository contains the Torb HTTP and WebSocket API. A frontend is not included.

## What Torb provides

### Accounts and authentication

- Email-and-password registration with email verification
- Password login, password reset, token refresh, and logout
- Google OpenID Connect login using state and PKCE
- Google account linking and the option for Google users to add password login
- RS256 access tokens and Redis-backed refresh-token rotation/revocation

Access tokens are valid for 15 minutes and are returned in the login response. Refresh tokens are valid for 7 days and are stored in a Secure, HTTP-only, SameSite=Lax cookie named `TOKEN`.

### Contributor profiles

- Profile summary with identity, skills, timezone, login methods, and activity counts
- Editable username, display name, skills, and timezone
- Avatar uploads to Amazon S3 with short-lived presigned URLs
- Counts for joined projects, assigned tasks, and completed tasks

### Projects and membership

- Create projects with a name, description, and desired skills
- Browse projects that the current user has not joined
- View task-status summaries for each project
- Request to join a project and track the request status
- Review and accept or reject join requests as the project owner
- View project members and recent created/joined projects

Creating a project automatically makes its creator an `Owner`. An accepted join request creates a `Member` membership.

### Tasks and collaboration

- Create tasks with initial assignees and a workflow status
- List project tasks and their assignees
- Update task title, description, and status
- Add or remove assignees
- Add and read task comments
- View recently assigned tasks and recent unassigned tasks from owned projects

Task statuses are `Unassigned`, `Ongoing`, `Completed`, and `Abandoned`. Project owners create tasks and manage assignees; owners and assigned contributors can update task details or status; all project members can read tasks and participate in comments.

### Notifications and realtime updates

Torb stores notifications in PostgreSQL, lets users mark them as read, and exposes a WebSocket channel for live activity. Events cover:

- Tasks added or updated
- Assignees added or removed
- Join requests created or answered
- Comments added

## Typical contribution flow

1. A user registers with a password or signs in with Google.
2. A project owner creates a project and adds tasks.
3. Another user discovers the project and submits a join request.
4. The owner accepts the request, making that user a project member.
5. The owner assigns tasks to members.
6. Owners and assignees update task progress, while members discuss work in comments.
7. Affected users receive persistent activity notifications and WebSocket events.

## Permissions at a glance

| Action | Who can perform it |
| --- | --- |
| Create or discover projects | Any authenticated user |
| Review and answer join requests | Project owner |
| List members, tasks, and comments | Project members, including the owner |
| Create tasks or change assignees | Project owner |
| Update a task's details or status | Project owner or a task assignee |
| Add comments | Project members, including the owner |

## Technology

| Area | Implementation |
| --- | --- |
| API | Go 1.25, `net/http`, Go 1.22-style `ServeMux` routing |
| Database | PostgreSQL with GORM |
| Ephemeral state | Redis for refresh tokens, OAuth state, and PKCE data |
| Authentication | RS256 JWT, bcrypt, Google OpenID Connect/OAuth 2.0 |
| Email | Resend |
| File storage | Amazon S3 |
| Realtime | WebSockets via `coder/websocket` |
| Tests | `testing`, Testify, and Testcontainers |

## Architecture

The application uses a small layered design:

```text
HTTP / WebSocket clients
          |
          v
api + middlewares        request parsing, responses, authentication
          |
          v
core services            business rules and permissions
          |
          v
repositories + models    PostgreSQL persistence through GORM

Supporting services: Redis, Google, Resend, Amazon S3, WebSocket notifier
```

The main packages are:

| Path | Responsibility |
| --- | --- |
| `cmd` | Configuration, dependency wiring, migrations, routes, and server startup |
| `api` | HTTP handlers and request/response types |
| `auth` | JWT, refresh-token storage, password auth, and Google OpenID Connect |
| `core` | Project, task, member, assignee, comment, join-request, and user rules |
| `models` | GORM persistence models |
| `middlewares` | Bearer authentication and HTTP error mapping |
| `notifications` | Persistent activity notification creation and retrieval |
| `realtime` | In-process WebSocket client registry and event delivery |
| `testhelpers`, `testdata` | Testcontainers setup, fixtures, and test migrations |

At startup, GORM creates or updates the schema and recreates a `project_summary` database view used for task-status counts.

## API overview

The REST API is served under:

```text
http://localhost:8080/api/v1
```

Protected endpoints expect:

```http
Authorization: Bearer <access-token>
```

The main route groups are:

- **Authentication:** `/auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout`, email verification, password reset, and `/auth/google/*`
- **Profile:** `/profile`, `/users`, `/profile/avatar`, `/profile/add-password`, and `/profile/google/*`
- **Projects:** `/projects`, `/public/projects`, `/projects/{id}/members`, `/projects/{id}/join-requests`, and project dashboard routes
- **Tasks and comments:** `/projects/{project_id}/tasks`, nested task/comment routes, and task dashboard routes
- **Notifications:** `/messages` and `/messages/{id}`

Successful HTTP responses use a common envelope:

```json
{
  "status": "success",
  "message": "optional message",
  "data": {}
}
```

Errors use the same `status` and `message` shape with the appropriate HTTP status code.

### WebSocket protocol

The WebSocket endpoint is the server root, outside the REST prefix:

```text
ws://localhost:8080/
```

Within 10 seconds of connecting, send the current access token:

```json
{"type":"token","token":"<access-token>"}
```

The server acknowledges authentication with `{"type":"ack"}`. When the access-token window expires, it sends `{"type":"refresh"}`; refresh the HTTP session and send the new access token over the socket.

## Local development

### Prerequisites

Install the following before starting:

- Go 1.25 or newer
- Docker Engine with Docker Compose v2
- OpenSSL
- A Resend API key for registration and password-reset email
- Google OAuth credentials for Google login and account linking
- AWS credentials and an S3 bucket named `torb` if testing avatar features

Docker Compose provisions PostgreSQL and Redis only. The Go API runs directly on the host, and this repository does not include a frontend.

### 1. Clone the repository and download dependencies

```bash
git clone https://github.com/Arup3201/torb.git
cd torb
go mod download
```

### 2. Start PostgreSQL and Redis

```bash
docker compose up -d pg redis
docker compose ps
```

The included Compose configuration exposes:

- PostgreSQL at `localhost:5432` with database `torb`, user `postgres`, and password `1234`
- Redis at `localhost:6379` without a password

PostgreSQL data is kept in the named `torb` volume between restarts.

### 3. Configure the environment

Create your local environment file:

```bash
cp .env.example .env
```

Then make sure `.env` contains every setting below. The application does not supply defaults, and `FRONTEND_URL` must be added if it is missing from the example file.

```dotenv
HOST=localhost
PORT=8080

DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASS=1234
DB_NAME=torb

REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASS=

RESEND_API_KEY=replace-with-your-resend-key

GOOGLE_CLIENT_ID=replace-with-your-google-client-id
GOOGLE_CLIENT_SECRET=replace-with-your-google-client-secret
GOOGLE_REDIRECT_URI=http://localhost:8080/api/v1/auth/google/callback
GOOGLE_CONNECT_REDIRECT_URI=http://localhost:8080/api/v1/profile/google/callback

FRONTEND_URL=http://localhost:3000

# Needed for avatar upload and avatar URL generation.
AWS_REGION=replace-with-your-bucket-region
AWS_ACCESS_KEY_ID=replace-with-your-access-key
AWS_SECRET_ACCESS_KEY=replace-with-your-secret-key
# AWS_SESSION_TOKEN=replace-if-using-temporary-credentials
```

Use `FRONTEND_URL` without a trailing slash. It is used as the allowed CORS origin and as the base for login, verification, and password-reset redirects. Register both Google callback URLs exactly as shown with your Google OAuth client.

All application-specific variables above are required to be nonempty at startup except `REDIS_PASS`. Nonempty placeholder values allow the server to start, but registration/email and Google flows require working Resend and Google credentials. AWS uses the standard SDK credential chain and is only exercised by avatar operations; the configured identity needs access to an existing S3 bucket named `torb`.

### 4. Generate the JWT signing key

From the repository root, generate the PKCS#8 RSA private key expected by the server:

```bash
openssl genpkey -algorithm RSA -out private.key -pkeyopt rsa_keygen_bits:2048
```

The server reads the fixed relative path `private.key`, so run it from the repository root. Both `.env` and `*.key` are ignored by Git.

### 5. Export the environment and run the API

The application reads process environment variables directly; it does not load `.env` itself.

```bash
set -a
source .env
set +a
go run ./cmd
```

On first startup, the server runs the database migrations automatically. A successful start prints a message similar to:

```text
[INFO] server starting at localhost:8080
```

There is no health-check route. To confirm the HTTP server is responding, request a protected endpoint without a token:

```bash
curl -i http://localhost:8080/api/v1/projects
```

An `HTTP/1.1 401 Unauthorized` response confirms that the server and authentication middleware are reachable.

The project has no hot-reload configuration, so restart `go run ./cmd` after code changes.

### 6. Stop the local services

Stop the API with `Ctrl+C`, then stop PostgreSQL and Redis:

```bash
docker compose down
```

This preserves the database volume. To intentionally delete all local PostgreSQL data and start with a clean database, use `docker compose down -v`.

## Running tests

Run the full suite from the repository root:

```bash
go test ./...
```

The integration tests use Testcontainers to create disposable PostgreSQL and Redis containers. The Docker daemon must therefore be running and accessible; the application `.env` and manually started Compose services are not needed. The first test run may download the container images. On a resource-constrained machine, run packages serially:

```bash
go test -p 1 ./...
```

## Current development notes

- The API accepts `page` and `limit` on some list routes, but repository-level pagination is not implemented yet.
- Project and task deletion, project updates, member removal, and comment editing/deletion are not currently exposed as routes.
- Realtime delivery uses an in-process WebSocket registry; it is not a distributed message bus.
- WebSocket origin verification is currently relaxed for development and should be hardened before production use.
- Refresh cookies are always marked `Secure`; local browser behavior over plain HTTP may affect refresh and logout testing.
- Resend sender addresses and the S3 bucket name are currently defined in code rather than configuration.

Additional implementation notes are available in [docs.mdx](docs.mdx).

## License

Torb is licensed under the [GNU General Public License v3.0](LICENSE).
