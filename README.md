# YAITracker

**Your Automated Intelligence Tracker** (or *Yet Another Issue Tracker*, depending on how many grey hairs you have).

A self-hosted, AI-native issue tracker with built-in [MCP](https://modelcontextprotocol.io/) integration, human/agent time tracking, and velocity analytics. An alternative to Jira and Linear for developers who work alongside AI agents.

> **Alpha Software** -- YAITracker is under active development. Features are missing, some things are broken, and the API will change. I'm dogfooding this project as I build it -- logging issues and tracking time from day one, then using what I find to improve it as I go. Send me your feedback about what you want to see in this project, your own AI workflows, or anything else that you think would be useful.

## Why YAITracker?

Traditional issue trackers were built for a world where humans write all the code. AI coding tools like Cursor, Copilot, and Claude have fundamentally changed development velocity, but project management hasn't caught up. When AI agents can produce code in minutes, the old assumptions about estimation, sprint planning, and time tracking fall apart.

The most effective developers in the AI age aren't just using one agent -- they're orchestrating multiple agents in parallel, sometimes across multiple projects at once. One person's hour of productivity can look radically different from another's depending on how well they manage that orchestration. The right tooling makes that difference easier to see, measure, and act on.

YAITracker is built around these ideas:

- **Automate the busywork of project management** -- Keeping issues updated with good notes, accurate time logs, and useful documentation was always valuable but painfully time-consuming. With MCP integration, your AI assistant handles that overhead at the speed of the conversation -- creating issues, logging time, adding summaries, moving cards -- so you get more consistent, more detailed tracking than ever with almost zero manual effort.
- **Understand the full picture of your output** -- See how much work was done overall and how much of it was a result of your effort. Clock in and out of work sessions, track agent timers alongside your own, and see the full scope of what you produced. When you're orchestrating multiple agents across multiple projects, this is how you measure and improve your real productivity.
- **AI as a first-class participant** -- A full [Model Context Protocol](https://modelcontextprotocol.io/) (MCP) server lets AI assistants in Cursor, Claude Desktop, or any MCP-compatible client create issues, start timers, move cards on the board, and log their own work -- all without leaving the editor. This also opens the door for teams working with project managers, stakeholders, and other AI-enhanced participants to collaborate efficiently through a shared, always-up-to-date project state.
- **Turn the data into better decisions** -- Track bugs, time, and issues across projects and tag them by framework, language, or workflow. Use velocity analytics, cross-project comparison, and prediction tools to understand which technologies you get the best results with, where the bottlenecks are, and how to estimate more accurately as your team (human + AI) evolves.

The goal is a full-featured issue tracker that helps people get more done, at higher quality, in the age of AI tools and orchestration. Right now it's in early development and best suited for small, personal deployments -- but the vision is much bigger than that.

## Features

- **Issue Tracking** -- Jira-style issues with types, priorities, labels, epics, and parent/child hierarchy
- **Kanban Board** -- Drag-and-drop board with real-time updates
- **Time Tracking** -- Real-time start/stop timers, work sessions (clock-in/out), human vs. agent time split
- **Velocity Analytics** -- Sprint velocity, cycle time, estimation accuracy, project health
- **Cross-Project Comparison** -- Compare metrics across projects by technology tags (Go vs. PHP, etc.)
- **Project Prediction** -- Estimate timelines for new projects based on historical data
- **MCP Server** -- Full MCP integration for AI-powered project management via tools and resources
- **REST API** -- JSON API with OAuth2 authentication for mobile apps and integrations
- **Web UI** -- Server-rendered HTML with htmx and Alpine.js (no SPA framework needed)
- **Simple Deployment** -- Single binary with embedded static assets. SQLite for zero-config local use today; additional database backends planned.

## Quick Start

### From Source

```bash
git clone https://github.com/loweryaustin/YAITracker.git
cd YAITracker
make build

export YAITRACKER_SECRET="your-secret-key-at-least-32-characters-long"

./yaitracker serve
# Open http://localhost:8080
```

### Docker

```bash
make docker

echo 'YAITRACKER_SECRET=your-secret-key-at-least-32-characters-long' > .env

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

## CLI

```bash
yaitracker serve --addr :8080 --db yaitracker.db

yaitracker mcp --db yaitracker.db
```

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--db` | `YAITRACKER_DB` | `yaitracker.db` | SQLite database file path |
| `--addr` | `YAITRACKER_ADDR` | `:8080` | HTTP listen address |
| `--secret` | `YAITRACKER_SECRET` | (required) | Application secret (32+ chars) |
| `--cors` | `YAITRACKER_CORS_ORIGINS` | (empty) | Allowed CORS origins for API |

## MCP Server

YAITracker includes a first-class MCP server so your AI assistant can manage issues, track time, and query analytics without leaving your editor. Connect it to any MCP-compatible client (Cursor, Claude Desktop, etc.).

### Setup in Cursor

Add to `.cursor/mcp.json`:

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

Or connect to a running instance over HTTP:

```json
{
  "mcpServers": {
    "yaitracker": {
      "url": "http://localhost:8080/mcp"
    }
  }
}
```

### Tools

| Tool | Description |
|------|-------------|
| `begin_work` | Start working on an issue (starts timer, moves to in-progress) |
| `complete_work` | Finish an issue (stops timer, adds summary, moves to done) |
| `start_timer` | Start an agent timer on an issue |
| `stop_timer` | Stop an active timer by ID or project key + number |
| `get_session_status` | Get current session, active timers, and utilization metrics |
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
| `get_time_entries` | Get time entries for an issue |
| `get_velocity` | Get velocity data for a project |
| `get_estimation_accuracy` | Get estimation accuracy report |
| `get_project_health` | Get project health summary |
| `compare_by_tag` | Compare project metrics by tag group |
| `predict_new_project` | Predict timeline for a new project based on historical data |

### Resources

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

## Architecture

```
yaitracker serve    → chi Router → HTML handlers (htmx/Alpine.js UI)
                                 → JSON API (/api/v1, OAuth2)
                                 → Static assets (embedded)

yaitracker mcp      → MCP Server (stdio or HTTP) → tools + resources
```

Both commands share the same SQLite database. The entire application compiles to a single binary with all assets embedded.

## Development

```bash
make dev          # Build and run with dev settings
make test         # Run tests
make vulncheck    # Check for vulnerabilities
make fmt          # Format code
make lint         # Lint
make clean        # Clean build artifacts
```

## Contributing

YAITracker is open to contributions. If you have ideas about how AI changes development workflows -- or you just want better issue tracking -- open an issue or submit a PR.

This project follows [Conventional Commits](https://www.conventionalcommits.org/) and [Gitflow](https://nvie.com/posts/a-successful-git-branching-model/) branching.

## License

[GNU Affero General Public License v3.0](LICENSE)
