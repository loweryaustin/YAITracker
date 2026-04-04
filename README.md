# YAITracker

A self-hosted issue tracker with built-in time tracking, velocity analytics, and a first-class MCP server for AI interaction. Ships as a single Go binary with embedded SQLite.

## Features

- **Issue Tracking** -- Jira-style issues with types, priorities, labels, epics, parent/child hierarchy
- **Kanban Board** -- Drag-and-drop board with SortableJS, real-time updates via htmx
- **Time Tracking** -- Real-time start/stop timers, work sessions (clock-in/out), human vs agent time split
- **Velocity Analytics** -- Sprint velocity, cycle time, estimation accuracy, project health
- **Cross-Project Comparison** -- Compare metrics across projects by technology tags (Go vs PHP, etc.)
- **Project Prediction** -- Estimate timelines for new projects based on historical data
- **MCP Server** -- Full MCP integration for AI-powered project management via tools and resources
- **REST API** -- JSON API with OAuth2 authentication for mobile apps and integrations
- **Web UI** -- Server-rendered HTML with htmx and Alpine.js (no SPA framework needed)

## Quick Start

### From Source

```bash
# Clone and build
git clone https://github.com/loweryaustin/YAITracker.git
cd YAITracker
make build

# Set a secret (must be 32+ characters)
export YAITRACKER_SECRET="your-secret-key-at-least-32-characters-long"

# Run the web server
./yaitracker serve

# Open http://localhost:8080
```

### Docker

```bash
# Build the image
make docker

# Create a .env file
echo 'YAITRACKER_SECRET=your-secret-key-at-least-32-characters-long' > .env

# Start the container
docker compose up -d

# Open http://localhost:8080
```

### Docker with Caddy (Production)

For production with automatic TLS, add a Caddy reverse proxy. Create `Caddyfile`:

```
yaitracker.example.com {
    reverse_proxy yaitracker:8080
}
```

Add Caddy to `docker-compose.yml`:

```yaml
services:
  caddy:
    image: caddy:2-alpine
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - caddy-data:/data
      - caddy-config:/config

volumes:
  caddy-data:
  caddy-config:
```

## CLI Usage

```bash
# Start the web server
yaitracker serve --addr :8080 --db yaitracker.db

# Start the MCP server (stdio transport)
yaitracker mcp --db yaitracker.db
```

### Flags

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--db` | `YAITRACKER_DB` | `yaitracker.db` | SQLite database file path |
| `--addr` | `YAITRACKER_ADDR` | `:8080` | HTTP listen address |
| `--secret` | `YAITRACKER_SECRET` | (required) | Application secret (32+ chars) |
| `--cors` | `YAITRACKER_CORS_ORIGINS` | (empty) | Allowed CORS origins for API |

## MCP Server

YAITracker includes a first-class MCP server for AI interaction. Connect it to any MCP-compatible client (Cursor, Claude Desktop, etc.).

### Setup in Cursor

Add to your MCP settings:

```json
{
  "mcpServers": {
    "yaitracker": {
      "command": "/path/to/yaitracker",
      "args": ["mcp", "--db", "/path/to/yaitracker.db"]
    }
  }
}
```

### Available Tools

| Tool | Description |
|------|-------------|
| `list_projects` | List all projects with summary stats |
| `create_project` | Create a new project |
| `delete_project` | Permanently delete a project and all related data |
| `tag_project` | Add or remove a tag on a project |
| `list_tags` | List all tags with usage counts |
| `list_issues` | List issues with optional filters (status, type, assignee, query) |
| `get_issue` | Get full issue detail with comments, time entries, and labels |
| `create_issue` | Create a new issue with type, priority, estimates |
| `update_issue` | Update issue fields |
| `move_issue` | Change issue status (move on the board) |
| `delete_issue` | Permanently delete an issue and all related data |
| `add_comment` | Add a comment to an issue |
| `search_issues` | Search issues across all projects |
| `start_timer` | Start an agent timer on an issue |
| `stop_timer` | Stop an active timer by ID or project key + number |
| `get_session_status` | Get current session, active timers, and utilization metrics |
| `get_time_entries` | Get time entries for an issue |
| `begin_work` | Start working on an issue (starts timer, moves to in-progress) |
| `complete_work` | Finish an issue (stops timer, adds summary, moves to done) |
| `get_velocity` | Get velocity data for a project |
| `get_estimation_accuracy` | Get estimation accuracy report |
| `get_project_health` | Get project health summary |
| `compare_by_tag` | Compare project metrics by tag group |
| `predict_new_project` | Predict timeline for a new project based on historical data |

### Available Resources

| Resource | Description |
|----------|-------------|
| `yaitracker://projects` | All projects with summary stats |
| `yaitracker://projects/{key}` | Project detail with health metrics |
| `yaitracker://projects/{key}/issues/{number}` | Issue detail with comments and time |
| `yaitracker://projects/{key}/board` | Kanban board state |
| `yaitracker://projects/{key}/velocity` | Velocity chart data |

## REST API

The JSON API lives at `/api/v1` and uses OAuth2 password grant for authentication.

### Authentication

```bash
# Get an access token
curl -X POST http://localhost:8080/api/v1/auth/token \
  -H "Content-Type: application/json" \
  -d '{"email": "you@example.com", "password": "your-password"}'

# Use the token
curl http://localhost:8080/api/v1/projects \
  -H "Authorization: Bearer <access_token>"

# Refresh the token
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "<refresh_token>"}'
```

### Endpoints

**Projects**: `GET/POST /projects`, `GET/PATCH/DELETE /projects/{key}`

**Issues**: `GET/POST /projects/{key}/issues`, `GET/PATCH/DELETE /projects/{key}/issues/{number}`

**Board**: `GET /projects/{key}/board`, `PATCH /projects/{key}/board/move`

**Comments**: `GET/POST /issues/{id}/comments`, `PATCH/DELETE /comments/{id}`

**Labels**: `GET/POST /projects/{key}/labels`, `PATCH/DELETE /labels/{id}`

**Tags**: `GET/POST /projects/{key}/tags`, `DELETE /projects/{key}/tags/{tag}`, `GET /tags`

**Time Tracking**: `POST /time/start`, `POST /time/stop`, `GET /time/active`, `GET/POST /issues/{id}/time`, `PATCH/DELETE /time/{id}`, `GET /time/sheet`

**Analytics**: `GET /projects/{key}/analytics/velocity`, `GET /analytics/compare`, `GET /analytics/predict`

## Security

YAITracker is designed for public internet exposure with multiple layers of defense:

- **Authentication**: bcrypt password hashing, account lockout after failed attempts, 12+ character passwords
- **Sessions**: Secure HttpOnly cookies, SameSite=Lax, server-side session storage with rotation
- **API Auth**: OAuth2 with access/refresh token rotation and breach detection
- **CSRF**: Double-submit cookie pattern on all mutating HTML endpoints
- **Rate Limiting**: Per-IP rate limiting on auth endpoints, per-user on API
- **Security Headers**: CSP, HSTS, X-Frame-Options, X-Content-Type-Options, Referrer-Policy
- **Input Validation**: Parameterized SQL queries, HTML sanitization on markdown
- **Docker**: Non-root user, read-only filesystem, all capabilities dropped, no privilege escalation

## Development

```bash
# Build and run with dev settings
make dev

# Run tests
make test

# Check for vulnerabilities
make vulncheck

# Format code
make fmt

# Lint
make lint

# Clean build artifacts
make clean
```

## Architecture

```
yaitracker serve    → chi Router → HTML handlers (htmx/Alpine.js UI)
                                 → JSON API (/api/v1, OAuth2)
                                 → Static assets (embedded)

yaitracker mcp      → MCP Server (stdio) → tools + resources
```

Both commands share the same SQLite database. The entire application compiles to a single binary with all assets embedded.

## License

[GNU Affero General Public License v3.0](LICENSE)
