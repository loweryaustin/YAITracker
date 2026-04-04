-- +goose Up

CREATE TABLE users (
    id              TEXT PRIMARY KEY,
    email           TEXT UNIQUE NOT NULL,
    name            TEXT NOT NULL,
    password        TEXT NOT NULL,
    role            TEXT NOT NULL DEFAULT 'member',
    failed_attempts INTEGER NOT NULL DEFAULT 0,
    locked_until    DATETIME,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sessions (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);

CREATE TABLE oauth_tokens (
    id                 TEXT PRIMARY KEY,
    user_id            TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    access_token       TEXT UNIQUE NOT NULL,
    refresh_token      TEXT UNIQUE NOT NULL,
    access_expires_at  DATETIME NOT NULL,
    refresh_expires_at DATETIME NOT NULL,
    client_name        TEXT,
    created_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_oauth_tokens_user ON oauth_tokens(user_id);
CREATE INDEX idx_oauth_tokens_access ON oauth_tokens(access_token);
CREATE INDEX idx_oauth_tokens_refresh ON oauth_tokens(refresh_token);

CREATE TABLE projects (
    id           TEXT PRIMARY KEY,
    key          TEXT UNIQUE NOT NULL,
    name         TEXT NOT NULL,
    description  TEXT,
    status       TEXT NOT NULL DEFAULT 'active',
    target_date  DATE,
    budget_hours REAL,
    created_by   TEXT NOT NULL REFERENCES users(id),
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE project_tags (
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    tag        TEXT NOT NULL,
    group_name TEXT,
    PRIMARY KEY (project_id, tag)
);

CREATE INDEX idx_project_tags_group ON project_tags(group_name);
CREATE INDEX idx_project_tags_tag ON project_tags(tag);

CREATE TABLE project_members (
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       TEXT NOT NULL DEFAULT 'member',
    PRIMARY KEY (project_id, user_id)
);

CREATE TABLE issues (
    id              TEXT PRIMARY KEY,
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    number          INTEGER NOT NULL,
    title           TEXT NOT NULL,
    description     TEXT,
    type            TEXT NOT NULL DEFAULT 'task',
    status          TEXT NOT NULL DEFAULT 'backlog',
    priority        TEXT NOT NULL DEFAULT 'none',
    assignee_id     TEXT REFERENCES users(id),
    reporter_id     TEXT NOT NULL REFERENCES users(id),
    parent_id       TEXT REFERENCES issues(id),
    sort_order      REAL NOT NULL DEFAULT 0,
    story_points    INTEGER,
    estimated_hours REAL,
    started_at      DATETIME,
    completed_at    DATETIME,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, number)
);

CREATE INDEX idx_issues_project ON issues(project_id);
CREATE INDEX idx_issues_status ON issues(project_id, status);
CREATE INDEX idx_issues_assignee ON issues(assignee_id);
CREATE INDEX idx_issues_parent ON issues(parent_id);

CREATE TABLE labels (
    id         TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    color      TEXT NOT NULL,
    UNIQUE(project_id, name)
);

CREATE TABLE issue_labels (
    issue_id TEXT NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    label_id TEXT NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
    PRIMARY KEY (issue_id, label_id)
);

CREATE TABLE comments (
    id         TEXT PRIMARY KEY,
    issue_id   TEXT NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    author_id  TEXT NOT NULL REFERENCES users(id),
    body       TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_comments_issue ON comments(issue_id);

CREATE TABLE time_entries (
    id          TEXT PRIMARY KEY,
    issue_id    TEXT NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    description TEXT,
    started_at  DATETIME NOT NULL,
    ended_at    DATETIME,
    duration    INTEGER,
    source      TEXT NOT NULL DEFAULT 'timer',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_time_entries_issue ON time_entries(issue_id);
CREATE INDEX idx_time_entries_user ON time_entries(user_id);
CREATE UNIQUE INDEX idx_active_timer ON time_entries(user_id) WHERE ended_at IS NULL;

CREATE TABLE activity_log (
    id          TEXT PRIMARY KEY,
    entity_type TEXT NOT NULL DEFAULT 'issue',
    entity_id   TEXT,
    user_id     TEXT NOT NULL REFERENCES users(id),
    action      TEXT NOT NULL,
    field       TEXT,
    old_value   TEXT,
    new_value   TEXT,
    ip_address  TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_activity_entity ON activity_log(entity_type, entity_id);
CREATE INDEX idx_activity_user ON activity_log(user_id, created_at);

-- +goose Down

DROP TABLE IF EXISTS activity_log;
DROP TABLE IF EXISTS time_entries;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS issue_labels;
DROP TABLE IF EXISTS labels;
DROP TABLE IF EXISTS issues;
DROP TABLE IF EXISTS project_members;
DROP TABLE IF EXISTS project_tags;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS oauth_tokens;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
