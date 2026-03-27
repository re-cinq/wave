package pipeline

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/relay"
)

// mockCompactionAdapter implements relay.CompactionAdapter for testing.
type mockCompactionAdapter struct {
	result string
	err    error
	called bool
}

func (m *mockCompactionAdapter) RunCompaction(_ context.Context, cfg relay.CompactionConfig) (string, error) {
	m.called = true
	return m.result, m.err
}

func TestNewThreadManager(t *testing.T) {
	tm := NewThreadManager(nil)
	if tm == nil {
		t.Fatal("expected non-nil ThreadManager")
	}
	if tm.maxTranscriptSize != defaultMaxTranscriptSize {
		t.Errorf("expected maxTranscriptSize=%d, got %d", defaultMaxTranscriptSize, tm.maxTranscriptSize)
	}
}

func TestThreadManager_AppendAndGetTranscript_Full(t *testing.T) {
	tm := NewThreadManager(nil)
	ctx := context.Background()

	tm.AppendTranscript("impl", "step-a", "I implemented the feature")
	tm.AppendTranscript("impl", "step-b", "I fixed the tests")

	result := tm.GetTranscript(ctx, "impl", FidelityFull)

	if !strings.Contains(result, "## Step: step-a at") {
		t.Error("expected step-a attribution header in full transcript")
	}
	if !strings.Contains(result, "I implemented the feature") {
		t.Error("expected step-a content in full transcript")
	}
	if !strings.Contains(result, "## Step: step-b at") {
		t.Error("expected step-b attribution header in full transcript")
	}
	if !strings.Contains(result, "I fixed the tests") {
		t.Error("expected step-b content in full transcript")
	}

	// Verify ordering: step-a before step-b
	posA := strings.Index(result, "step-a")
	posB := strings.Index(result, "step-b")
	if posA >= posB {
		t.Errorf("expected step-a before step-b in transcript, posA=%d posB=%d", posA, posB)
	}
}

func TestThreadManager_GetTranscript_Compact(t *testing.T) {
	tm := NewThreadManager(nil)
	ctx := context.Background()

	longContent := strings.Repeat("x", 1000)
	tm.AppendTranscript("impl", "step-a", longContent)

	result := tm.GetTranscript(ctx, "impl", FidelityCompact)

	if !strings.Contains(result, "### step-a (completed)") {
		t.Error("expected compact header for step-a")
	}
	// Content should be truncated to 500 chars + "..."
	if strings.Contains(result, longContent) {
		t.Error("compact fidelity should truncate long content")
	}
	if !strings.Contains(result, "...") {
		t.Error("expected truncation marker ...")
	}
}

func TestThreadManager_GetTranscript_Fresh(t *testing.T) {
	tm := NewThreadManager(nil)
	ctx := context.Background()

	tm.AppendTranscript("impl", "step-a", "some output")

	result := tm.GetTranscript(ctx, "impl", FidelityFresh)
	if result != "" {
		t.Errorf("expected empty string for fresh fidelity, got %q", result)
	}
}

func TestThreadManager_GetTranscript_Summary_WithAdapter(t *testing.T) {
	mock := &mockCompactionAdapter{result: "LLM summary of conversation"}
	tm := NewThreadManager(mock)
	ctx := context.Background()

	tm.AppendTranscript("impl", "step-a", "I implemented feature X")

	result := tm.GetTranscript(ctx, "impl", FidelitySummary)
	if result != "LLM summary of conversation" {
		t.Errorf("expected LLM summary, got %q", result)
	}
	if !mock.called {
		t.Error("expected compaction adapter to be called")
	}
}

func TestThreadManager_GetTranscript_Summary_FallbackToCompact(t *testing.T) {
	mock := &mockCompactionAdapter{err: fmt.Errorf("compaction failed")}
	tm := NewThreadManager(mock)
	ctx := context.Background()

	tm.AppendTranscript("impl", "step-a", "output")

	result := tm.GetTranscript(ctx, "impl", FidelitySummary)
	// Should fall back to compact format
	if !strings.Contains(result, "### step-a (completed)") {
		t.Errorf("expected compact fallback on compaction error, got %q", result)
	}
}

func TestThreadManager_GetTranscript_Summary_NilAdapter(t *testing.T) {
	tm := NewThreadManager(nil)
	ctx := context.Background()

	tm.AppendTranscript("impl", "step-a", "output")

	result := tm.GetTranscript(ctx, "impl", FidelitySummary)
	// Should fall back to compact format when no adapter
	if !strings.Contains(result, "### step-a (completed)") {
		t.Errorf("expected compact fallback with nil adapter, got %q", result)
	}
}

func TestThreadManager_GetTranscript_EmptyThread(t *testing.T) {
	tm := NewThreadManager(nil)
	ctx := context.Background()

	result := tm.GetTranscript(ctx, "nonexistent", FidelityFull)
	if result != "" {
		t.Errorf("expected empty string for unknown thread, got %q", result)
	}
}

func TestThreadManager_TranscriptSizeCap(t *testing.T) {
	tm := NewThreadManager(nil)
	tm.maxTranscriptSize = 100 // Low cap for testing

	// Add entries that exceed the cap
	tm.AppendTranscript("impl", "step-1", strings.Repeat("a", 60))
	tm.AppendTranscript("impl", "step-2", strings.Repeat("b", 60))

	// Total would be 120 chars, exceeding 100 cap
	// Oldest entry (step-1) should be trimmed
	ctx := context.Background()
	result := tm.GetTranscript(ctx, "impl", FidelityFull)

	if strings.Contains(result, "step-1") {
		t.Error("expected oldest entry (step-1) to be trimmed after cap exceeded")
	}
	if !strings.Contains(result, "step-2") {
		t.Error("expected newest entry (step-2) to be preserved")
	}
}

func TestThreadManager_ThreadIsolation(t *testing.T) {
	tm := NewThreadManager(nil)
	ctx := context.Background()

	tm.AppendTranscript("thread-a", "step-1", "content for thread A")
	tm.AppendTranscript("thread-b", "step-2", "content for thread B")

	resultA := tm.GetTranscript(ctx, "thread-a", FidelityFull)
	resultB := tm.GetTranscript(ctx, "thread-b", FidelityFull)

	if strings.Contains(resultA, "content for thread B") {
		t.Error("thread-a should not contain thread-b content")
	}
	if strings.Contains(resultB, "content for thread A") {
		t.Error("thread-b should not contain thread-a content")
	}
}

func TestThreadManager_MultipleEntriesSameThread(t *testing.T) {
	tm := NewThreadManager(nil)
	ctx := context.Background()

	tm.AppendTranscript("impl", "step-1", "first")
	tm.AppendTranscript("impl", "step-2", "second")
	tm.AppendTranscript("impl", "step-3", "third")

	result := tm.GetTranscript(ctx, "impl", FidelityFull)

	// All entries should be present in order
	pos1 := strings.Index(result, "step-1")
	pos2 := strings.Index(result, "step-2")
	pos3 := strings.Index(result, "step-3")

	if pos1 >= pos2 || pos2 >= pos3 {
		t.Errorf("expected entries in order: step-1 (%d), step-2 (%d), step-3 (%d)", pos1, pos2, pos3)
	}
}

func TestThreadManager_ThreadSize(t *testing.T) {
	tm := NewThreadManager(nil)

	tm.AppendTranscript("impl", "step-1", "hello") // 5 chars
	tm.AppendTranscript("impl", "step-2", "world") // 5 chars

	size := tm.ThreadSize("impl")
	if size != 10 {
		t.Errorf("expected thread size 10, got %d", size)
	}

	emptySize := tm.ThreadSize("nonexistent")
	if emptySize != 0 {
		t.Errorf("expected size 0 for nonexistent thread, got %d", emptySize)
	}
}

func TestThreadManager_CompactShortContent(t *testing.T) {
	tm := NewThreadManager(nil)
	ctx := context.Background()

	// Content shorter than compactTruncateLen should not be truncated
	shortContent := "This is a short message."
	tm.AppendTranscript("impl", "step-a", shortContent)

	result := tm.GetTranscript(ctx, "impl", FidelityCompact)
	if strings.Contains(result, "...") {
		t.Error("short content should not be truncated")
	}
	if !strings.Contains(result, shortContent) {
		t.Error("expected full short content in compact output")
	}
}
