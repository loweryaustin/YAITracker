package store

import (
	"context"
	"database/sql"
	"time"

	"yaitracker.com/loweryaustin/internal/model"
)

func (s *Store) LogActivity(ctx context.Context, entry *model.ActivityLog) error {
	return s.writeTx(ctx, func(tx *sql.Tx) error {
		entry.ID = NewID()
		entry.CreatedAt = time.Now().UTC()

		_, err := tx.ExecContext(ctx,
			`INSERT INTO activity_log (id, entity_type, entity_id, user_id, action, field, old_value, new_value, ip_address, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			entry.ID, entry.EntityType, entry.EntityID, entry.UserID, entry.Action,
			entry.Field, entry.OldValue, entry.NewValue, entry.IPAddress, entry.CreatedAt,
		)
		return err
	})
}

func (s *Store) ListActivity(ctx context.Context, entityType, entityID string, limit, offset int) ([]model.ActivityLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var rows *sql.Rows
	var err error

	if entityType != "" && entityID != "" {
		rows, err = s.db.QueryContext(ctx,
			`SELECT a.id, a.entity_type, a.entity_id, a.user_id, a.action, a.field,
			        a.old_value, a.new_value, a.ip_address, a.created_at,
			        u.id, u.name, u.email
			 FROM activity_log a JOIN users u ON a.user_id = u.id
			 WHERE a.entity_type = ? AND a.entity_id = ?
			 ORDER BY a.created_at DESC LIMIT ? OFFSET ?`,
			entityType, entityID, limit, offset)
	} else {
		rows, err = s.db.QueryContext(ctx,
			`SELECT a.id, a.entity_type, a.entity_id, a.user_id, a.action, a.field,
			        a.old_value, a.new_value, a.ip_address, a.created_at,
			        u.id, u.name, u.email
			 FROM activity_log a JOIN users u ON a.user_id = u.id
			 ORDER BY a.created_at DESC LIMIT ? OFFSET ?`,
			limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []model.ActivityLog
	for rows.Next() {
		var a model.ActivityLog
		var u model.User
		var entityID, field, oldVal, newVal, ip sql.NullString

		if err := rows.Scan(&a.ID, &a.EntityType, &entityID, &a.UserID, &a.Action, &field,
			&oldVal, &newVal, &ip, &a.CreatedAt,
			&u.ID, &u.Name, &u.Email); err != nil {
			return nil, err
		}
		if entityID.Valid {
			a.EntityID = entityID.String
		}
		if field.Valid {
			a.Field = field.String
		}
		if oldVal.Valid {
			a.OldValue = oldVal.String
		}
		if newVal.Valid {
			a.NewValue = newVal.String
		}
		if ip.Valid {
			a.IPAddress = ip.String
		}
		a.User = &u
		activities = append(activities, a)
	}
	return activities, rows.Err()
}

func (s *Store) ListRecentActivity(ctx context.Context, limit int) ([]model.ActivityLog, error) {
	return s.ListActivity(ctx, "", "", limit, 0)
}

func (s *Store) ListProjectActivity(ctx context.Context, projectID string, limit, offset int) ([]model.ActivityLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT a.id, a.entity_type, a.entity_id, a.user_id, a.action, a.field,
		        a.old_value, a.new_value, a.ip_address, a.created_at,
		        u.id, u.name, u.email
		 FROM activity_log a JOIN users u ON a.user_id = u.id
		 WHERE (a.entity_type = 'issue' AND a.entity_id IN (SELECT id FROM issues WHERE project_id = ?))
		    OR (a.entity_type = 'project' AND a.entity_id = ?)
		 ORDER BY a.created_at DESC LIMIT ? OFFSET ?`,
		projectID, projectID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []model.ActivityLog
	for rows.Next() {
		var a model.ActivityLog
		var u model.User
		var eID, field, oldVal, newVal, ip sql.NullString

		if err := rows.Scan(&a.ID, &a.EntityType, &eID, &a.UserID, &a.Action, &field,
			&oldVal, &newVal, &ip, &a.CreatedAt,
			&u.ID, &u.Name, &u.Email); err != nil {
			return nil, err
		}
		if eID.Valid {
			a.EntityID = eID.String
		}
		if field.Valid {
			a.Field = field.String
		}
		if oldVal.Valid {
			a.OldValue = oldVal.String
		}
		if newVal.Valid {
			a.NewValue = newVal.String
		}
		if ip.Valid {
			a.IPAddress = ip.String
		}
		a.User = &u
		activities = append(activities, a)
	}
	return activities, rows.Err()
}
