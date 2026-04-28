package state

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// CreateWebhook inserts a new webhook and returns its ID.
func (s *stateStore) CreateWebhook(webhook *Webhook) (int64, error) {
	eventsJSON, err := json.Marshal(webhook.Events)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal webhook events: %w", err)
	}
	headersJSON, err := json.Marshal(webhook.Headers)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal webhook headers: %w", err)
	}

	now := time.Now()
	result, err := s.db.Exec(
		`INSERT INTO webhooks (name, url, events, matcher, headers, secret, active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		webhook.Name, webhook.URL, string(eventsJSON), webhook.Matcher,
		string(headersJSON), webhook.Secret, webhook.Active, now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create webhook: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get webhook ID: %w", err)
	}

	webhook.ID = id
	webhook.CreatedAt = now
	webhook.UpdatedAt = now
	return id, nil
}

// ListWebhooks returns all registered webhooks.
func (s *stateStore) ListWebhooks() ([]*Webhook, error) {
	rows, err := s.db.Query(
		`SELECT id, name, url, events, matcher, headers, secret, active, created_at, updated_at
		FROM webhooks ORDER BY id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}
	defer rows.Close()

	return scanWebhookRows(rows)
}

// GetWebhook retrieves a webhook by ID.
func (s *stateStore) GetWebhook(id int64) (*Webhook, error) {
	row := s.db.QueryRow(
		`SELECT id, name, url, events, matcher, headers, secret, active, created_at, updated_at
		FROM webhooks WHERE id = ?`,
		id,
	)

	w, err := scanWebhookRow(row)
	if err != nil {
		return nil, fmt.Errorf("webhook not found (id=%d): %w", id, err)
	}
	return w, nil
}

// UpdateWebhook updates an existing webhook.
func (s *stateStore) UpdateWebhook(webhook *Webhook) error {
	eventsJSON, err := json.Marshal(webhook.Events)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook events: %w", err)
	}
	headersJSON, err := json.Marshal(webhook.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook headers: %w", err)
	}

	now := time.Now()
	result, err := s.db.Exec(
		`UPDATE webhooks SET name = ?, url = ?, events = ?, matcher = ?, headers = ?, secret = ?, active = ?, updated_at = ?
		WHERE id = ?`,
		webhook.Name, webhook.URL, string(eventsJSON), webhook.Matcher,
		string(headersJSON), webhook.Secret, webhook.Active, now, webhook.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update webhook: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("webhook not found (id=%d)", webhook.ID)
	}

	webhook.UpdatedAt = now
	return nil
}

// DeleteWebhook removes a webhook by ID. Associated deliveries are cascade-deleted.
func (s *stateStore) DeleteWebhook(id int64) error {
	result, err := s.db.Exec("DELETE FROM webhooks WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete result: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("webhook not found (id=%d)", id)
	}
	return nil
}

// RecordWebhookDelivery records a webhook delivery attempt.
func (s *stateStore) RecordWebhookDelivery(delivery *WebhookDelivery) error {
	now := time.Now()
	result, err := s.db.Exec(
		`INSERT INTO webhook_deliveries (webhook_id, run_id, event, status_code, response_time_ms, error, delivered_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		delivery.WebhookID, delivery.RunID, delivery.Event,
		delivery.StatusCode, delivery.ResponseTimeMs, delivery.Error, now,
	)
	if err != nil {
		return fmt.Errorf("failed to record webhook delivery: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get delivery ID: %w", err)
	}

	delivery.ID = id
	delivery.DeliveredAt = now
	return nil
}

// GetWebhookDeliveries retrieves delivery records for a webhook, most recent first.
func (s *stateStore) GetWebhookDeliveries(webhookID int64, limit int) ([]*WebhookDelivery, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.Query(
		`SELECT id, webhook_id, run_id, event, status_code, response_time_ms, error, delivered_at
		FROM webhook_deliveries WHERE webhook_id = ?
		ORDER BY delivered_at DESC LIMIT ?`,
		webhookID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query webhook deliveries: %w", err)
	}
	defer rows.Close()

	var deliveries []*WebhookDelivery
	for rows.Next() {
		var d WebhookDelivery
		var deliveredAt time.Time
		var errStr sql.NullString
		var statusCode sql.NullInt64
		var responseTimeMs sql.NullInt64

		err := rows.Scan(&d.ID, &d.WebhookID, &d.RunID, &d.Event,
			&statusCode, &responseTimeMs, &errStr, &deliveredAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook delivery: %w", err)
		}

		d.DeliveredAt = deliveredAt
		if statusCode.Valid {
			d.StatusCode = int(statusCode.Int64)
		}
		if responseTimeMs.Valid {
			d.ResponseTimeMs = responseTimeMs.Int64
		}
		if errStr.Valid {
			d.Error = errStr.String
		}
		deliveries = append(deliveries, &d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating webhook deliveries: %w", err)
	}
	return deliveries, nil
}

// scanWebhookRow scans a single webhook row.
func scanWebhookRow(row *sql.Row) (*Webhook, error) {
	var w Webhook
	var eventsJSON, headersJSON string
	var active int

	err := row.Scan(&w.ID, &w.Name, &w.URL, &eventsJSON, &w.Matcher,
		&headersJSON, &w.Secret, &active, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, err
	}

	w.Active = active != 0

	if err := json.Unmarshal([]byte(eventsJSON), &w.Events); err != nil {
		return nil, fmt.Errorf("failed to unmarshal webhook events: %w", err)
	}
	if w.Events == nil {
		w.Events = []string{}
	}

	if err := json.Unmarshal([]byte(headersJSON), &w.Headers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal webhook headers: %w", err)
	}
	if w.Headers == nil {
		w.Headers = map[string]string{}
	}

	return &w, nil
}

// scanWebhookRows scans multiple webhook rows.
func scanWebhookRows(rows *sql.Rows) ([]*Webhook, error) {
	var webhooks []*Webhook
	for rows.Next() {
		var w Webhook
		var eventsJSON, headersJSON string
		var active int

		err := rows.Scan(&w.ID, &w.Name, &w.URL, &eventsJSON, &w.Matcher,
			&headersJSON, &w.Secret, &active, &w.CreatedAt, &w.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook row: %w", err)
		}

		w.Active = active != 0

		if err := json.Unmarshal([]byte(eventsJSON), &w.Events); err != nil {
			return nil, fmt.Errorf("failed to unmarshal webhook events: %w", err)
		}
		if w.Events == nil {
			w.Events = []string{}
		}

		if err := json.Unmarshal([]byte(headersJSON), &w.Headers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal webhook headers: %w", err)
		}
		if w.Headers == nil {
			w.Headers = map[string]string{}
		}

		webhooks = append(webhooks, &w)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating webhook rows: %w", err)
	}
	return webhooks, nil
}
