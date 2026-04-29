package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_FieldShapes(t *testing.T) {
	tests := []struct {
		name string
		expr string
		want bool // true => parse must succeed
	}{
		{"every minute", "* * * * *", true},
		{"top of hour", "0 * * * *", true},
		{"daily midnight", "0 0 * * *", true},
		{"weekday 9am", "0 9 * * 1-5", true},
		{"every 15 min", "*/15 * * * *", true},
		{"list", "0,15,30,45 * * * *", true},
		{"range/step", "0-30/5 * * * *", true},
		{"too few fields", "* * * *", false},
		{"too many fields", "* * * * * *", false},
		{"out of range", "60 * * * *", false},
		{"inverted range", "5-3 * * * *", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.expr)
			if tc.want && err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tc.expr, err)
			}
			if !tc.want && err == nil {
				t.Fatalf("Parse(%q) expected error, got nil", tc.expr)
			}
		})
	}
}

func TestNextFire_TopOfHour(t *testing.T) {
	expr, err := Parse("0 * * * *")
	require.NoError(t, err)
	now := time.Date(2026, 4, 29, 14, 27, 0, 0, time.UTC)
	got, err := expr.NextFire(now)
	require.NoError(t, err)
	want := time.Date(2026, 4, 29, 15, 0, 0, 0, time.UTC)
	assert.Equal(t, want, got)
}

func TestNextFire_Every15Min(t *testing.T) {
	expr, err := Parse("*/15 * * * *")
	require.NoError(t, err)
	now := time.Date(2026, 4, 29, 14, 7, 0, 0, time.UTC)
	got, err := expr.NextFire(now)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 4, 29, 14, 15, 0, 0, time.UTC), got)
}

func TestNextFire_WeekdayOnly(t *testing.T) {
	// Mon-Fri at 9am. Saturday's next fire is Monday morning.
	expr, err := Parse("0 9 * * 1-5")
	require.NoError(t, err)
	saturday := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC) // Saturday
	got, err := expr.NextFire(saturday)
	require.NoError(t, err)
	assert.Equal(t, time.Weekday(time.Monday), got.Weekday())
	assert.Equal(t, 9, got.Hour())
}

func TestNextFire_DomDowOr(t *testing.T) {
	// cron(8): non-* in BOTH dom and dow → match if EITHER fires.
	// "0 0 1 * 1" = midnight on the 1st OR every Monday.
	expr, err := Parse("0 0 1 * 1")
	require.NoError(t, err)
	// Friday Apr 24 → next match is Sun Apr 26? No, Apr 26 is Sunday.
	// First Monday after Friday Apr 24 is Mon Apr 27. The 1st is May 1.
	// So next from Apr 24 is Mon Apr 27.
	from := time.Date(2026, 4, 24, 0, 0, 0, 0, time.UTC)
	got, err := expr.NextFire(from)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC), got)
}

func TestNextFire_AlwaysStrictlyAfterNow(t *testing.T) {
	expr, err := Parse("* * * * *")
	require.NoError(t, err)
	now := time.Date(2026, 4, 29, 14, 27, 0, 0, time.UTC)
	got, err := expr.NextFire(now)
	require.NoError(t, err)
	assert.True(t, got.After(now), "NextFire must be strictly after now")
	assert.Equal(t, now.Add(time.Minute), got)
}

func TestNextFire_ImpossibleExpression(t *testing.T) {
	// Feb 31 — cannot ever match.
	expr, err := Parse("0 0 31 2 *")
	require.NoError(t, err)
	_, err = expr.NextFire(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	assert.Error(t, err)
}

func TestMatch_FieldRespect(t *testing.T) {
	expr, err := Parse("30 14 * * *")
	require.NoError(t, err)
	assert.True(t, expr.Match(time.Date(2026, 4, 29, 14, 30, 0, 0, time.UTC)))
	assert.False(t, expr.Match(time.Date(2026, 4, 29, 14, 31, 0, 0, time.UTC)))
	assert.False(t, expr.Match(time.Date(2026, 4, 29, 15, 30, 0, 0, time.UTC)))
}
