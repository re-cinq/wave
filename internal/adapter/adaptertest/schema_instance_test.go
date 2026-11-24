package adaptertest

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/recinq/wave/internal/adapter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSchema = `{
  "type": "object",
  "required": ["summary", "score", "items", "status", "meta"],
  "properties": {
    "summary": {"type": "string", "minLength": 20},
    "score": {"type": "integer", "minimum": 3, "maximum": 10},
    "items": {"type": "array", "minItems": 2, "items": {"$ref": "#/definitions/item"}},
    "status": {"enum": ["pass", "fail"]},
    "meta": {"type": "object", "properties": {"ok": {"type": "boolean"}}}
  },
  "definitions": {
    "item": {"type": "object", "required": ["id"], "properties": {"id": {"type": "string"}}}
  }
}`

func TestMockAdapter_SatisfiesPromptSchema(t *testing.T) {
	prompt := "Do the task.\n\n### Contract Schema\n\n" + schemaPromptMarker + testSchema + "\n```\n\n"
	res, err := NewMockAdapter().Run(context.Background(), adapter.AdapterRunConfig{Prompt: prompt, Persona: "navigator"})
	require.NoError(t, err)
	out, err := io.ReadAll(res.Stdout)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(out, &doc))
	assert.GreaterOrEqual(t, len(doc["summary"].(string)), 20)
	assert.Equal(t, float64(3), doc["score"])
	items := doc["items"].([]any)
	require.Len(t, items, 2)
	assert.Equal(t, "mock output", items[0].(map[string]any)["id"])
	assert.Equal(t, "pass", doc["status"])
	assert.Equal(t, true, doc["meta"].(map[string]any)["ok"])
}

func TestMockAdapter_NoSchemaFallsBack(t *testing.T) {
	assert.Nil(t, extractPromptSchema("no schema here"))
	assert.Nil(t, extractPromptSchema(schemaPromptMarker+"{not json\n```"))
}
