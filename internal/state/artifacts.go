package state

import (
	"database/sql"
	"fmt"
	"time"
)

func (s *stateStore) RegisterArtifact(runID string, stepID string, name string, path string, artifactType string, sizeBytes int64) error {
	now := time.Now().Unix()

	query := `INSERT INTO artifact (run_id, step_id, name, path, type, size_bytes, created_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query, runID, stepID, name, path, artifactType, sizeBytes, now)
	if err != nil {
		return fmt.Errorf("failed to register artifact: %w", err)
	}

	return nil
}

// GetArtifacts retrieves artifacts for a run, optionally filtered by step ID.
func (s *stateStore) GetArtifacts(runID string, stepID string) ([]ArtifactRecord, error) {
	query := `SELECT id, run_id, step_id, name, path, type, size_bytes, created_at
	          FROM artifact
	          WHERE run_id = ?`
	args := []any{runID}

	if stepID != "" {
		query += " AND step_id = ?"
		args = append(args, stepID)
	}

	query += " ORDER BY created_at ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query artifacts: %w", err)
	}
	defer rows.Close()

	var records []ArtifactRecord
	for rows.Next() {
		var record ArtifactRecord
		var createdAt int64
		var artifactType sql.NullString
		var sizeBytes sql.NullInt64

		err := rows.Scan(
			&record.ID,
			&record.RunID,
			&record.StepID,
			&record.Name,
			&record.Path,
			&artifactType,
			&sizeBytes,
			&createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan artifact: %w", err)
		}

		record.CreatedAt = time.Unix(createdAt, 0)
		if artifactType.Valid {
			record.Type = artifactType.String
		}
		if sizeBytes.Valid {
			record.SizeBytes = sizeBytes.Int64
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating artifacts: %w", err)
	}

	return records, nil
}

// RequestCancellation sets a cancellation flag for a run.

func (s *stateStore) SaveArtifactMetadata(artifactID int64, runID string, stepID string, previewText string, mimeType string, encoding string, metadataJSON string) error {
	now := time.Now().Unix()

	query := `INSERT INTO artifact_metadata (
	              artifact_id, run_id, step_id, preview_text, mime_type,
	              encoding, metadata_json, indexed_at
	          ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	          ON CONFLICT(artifact_id) DO UPDATE SET
	              preview_text = excluded.preview_text,
	              mime_type = excluded.mime_type,
	              encoding = excluded.encoding,
	              metadata_json = excluded.metadata_json,
	              indexed_at = excluded.indexed_at`

	_, err := s.db.Exec(query, artifactID, runID, stepID, previewText, mimeType, encoding, metadataJSON, now)
	if err != nil {
		return fmt.Errorf("failed to save artifact metadata: %w", err)
	}

	return nil
}

// GetArtifactMetadata retrieves extended metadata for an artifact.
func (s *stateStore) GetArtifactMetadata(artifactID int64) (*ArtifactMetadataRecord, error) {
	query := `SELECT artifact_id, run_id, step_id, preview_text, mime_type,
	                 encoding, metadata_json, indexed_at
	          FROM artifact_metadata
	          WHERE artifact_id = ?`

	var record ArtifactMetadataRecord
	var indexedAt int64
	var previewText, mimeType, encoding, metadataJSON sql.NullString

	err := s.db.QueryRow(query, artifactID).Scan(
		&record.ArtifactID,
		&record.RunID,
		&record.StepID,
		&previewText,
		&mimeType,
		&encoding,
		&metadataJSON,
		&indexedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("artifact metadata not found: %d", artifactID)
		}
		return nil, fmt.Errorf("failed to get artifact metadata: %w", err)
	}

	if previewText.Valid {
		record.PreviewText = previewText.String
	}
	if mimeType.Valid {
		record.MimeType = mimeType.String
	}
	if encoding.Valid {
		record.Encoding = encoding.String
	}
	if metadataJSON.Valid {
		record.MetadataJSON = metadataJSON.String
	}
	record.IndexedAt = time.Unix(indexedAt, 0)

	return &record, nil
}
