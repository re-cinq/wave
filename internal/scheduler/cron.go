// Package scheduler provides cron-driven recurring pipeline runs (epic
// #1565 PRE-6). The cron parser supports the standard 5-field expression
// (minute hour day-of-month month day-of-week) plus the common shortcuts
// `*`, `N`, `N-M`, `*/N`, and `N,M,P`. Day-of-month and day-of-week are
// OR'd when both are non-`*` — matching cron(8) semantics.
//
// NextFire walks forward minute-by-minute from `now` and returns the first
// matching tick. Capped at 5 years to surface impossible expressions
// (e.g. `0 0 31 2 *`) as errors rather than infinite loops.
package scheduler

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Expression is a parsed 5-field cron expression. Each field is a bitmap
// over the allowed range for that field; bit i is set when value i matches.
type Expression struct {
	raw     string
	minute  uint64 // bits 0..59
	hour    uint64 // bits 0..23
	dom     uint64 // bits 1..31; bit 0 unused
	month   uint64 // bits 1..12; bit 0 unused
	dow     uint64 // bits 0..6 (Sun=0, Sat=6)
	domStar bool   // dom field was `*`
	dowStar bool   // dow field was `*`
}

// Parse parses a 5-field cron expression. Whitespace between fields is
// collapsed; leading/trailing whitespace is trimmed.
func Parse(expr string) (*Expression, error) {
	fields := strings.Fields(strings.TrimSpace(expr))
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron: want 5 fields (min hour dom month dow), got %d in %q", len(fields), expr)
	}
	e := &Expression{raw: expr}
	var err error
	if e.minute, err = parseField(fields[0], 0, 59); err != nil {
		return nil, fmt.Errorf("cron: minute: %w", err)
	}
	if e.hour, err = parseField(fields[1], 0, 23); err != nil {
		return nil, fmt.Errorf("cron: hour: %w", err)
	}
	if e.dom, err = parseField(fields[2], 1, 31); err != nil {
		return nil, fmt.Errorf("cron: day-of-month: %w", err)
	}
	if e.month, err = parseField(fields[3], 1, 12); err != nil {
		return nil, fmt.Errorf("cron: month: %w", err)
	}
	if e.dow, err = parseField(fields[4], 0, 6); err != nil {
		return nil, fmt.Errorf("cron: day-of-week: %w", err)
	}
	e.domStar = fields[2] == "*"
	e.dowStar = fields[4] == "*"
	return e, nil
}

// String returns the original expression text for diagnostics.
func (e *Expression) String() string { return e.raw }

// Match reports whether t satisfies every field of e. cron(8) semantics:
// when both day-of-month and day-of-week are explicit (non-`*`), the
// expression matches if EITHER condition holds; otherwise both must hold.
func (e *Expression) Match(t time.Time) bool {
	if !bit(e.minute, t.Minute()) {
		return false
	}
	if !bit(e.hour, t.Hour()) {
		return false
	}
	if !bit(e.month, int(t.Month())) {
		return false
	}
	domHit := bit(e.dom, t.Day())
	dowHit := bit(e.dow, int(t.Weekday()))
	switch {
	case e.domStar && e.dowStar:
		return true
	case e.domStar:
		return dowHit
	case e.dowStar:
		return domHit
	default:
		return domHit || dowHit
	}
}

// NextFire returns the first minute strictly after `now` that satisfies
// the expression. Returns an error when no match is found within 5 years —
// surfaces impossible expressions like `0 0 31 2 *` (Feb 31).
func (e *Expression) NextFire(now time.Time) (time.Time, error) {
	// Truncate to minute resolution and step one minute forward so the
	// returned time is strictly greater than `now`.
	t := now.Truncate(time.Minute).Add(time.Minute)
	limit := t.Add(5 * 365 * 24 * time.Hour)
	for t.Before(limit) {
		if e.Match(t) {
			return t, nil
		}
		t = t.Add(time.Minute)
	}
	return time.Time{}, fmt.Errorf("cron: no match within 5 years for %q", e.raw)
}

// parseField parses a single cron field into a bitmap. The accepted forms
// (per the package doc) compose: a comma-separated list of items, each of
// which is `*`, `N`, `N-M`, `*/N`, or `N-M/N`.
func parseField(spec string, lo, hi int) (uint64, error) {
	if spec == "" {
		return 0, errors.New("empty field")
	}
	var bits uint64
	for _, item := range strings.Split(spec, ",") {
		stepBits, err := parseFieldItem(item, lo, hi)
		if err != nil {
			return 0, err
		}
		bits |= stepBits
	}
	if bits == 0 {
		return 0, fmt.Errorf("no values in %q", spec)
	}
	return bits, nil
}

func parseFieldItem(item string, lo, hi int) (uint64, error) {
	step := 1
	if i := strings.Index(item, "/"); i >= 0 {
		s, err := strconv.Atoi(item[i+1:])
		if err != nil || s < 1 {
			return 0, fmt.Errorf("invalid step in %q", item)
		}
		step = s
		item = item[:i]
	}

	var rangeLo, rangeHi int
	switch {
	case item == "*":
		rangeLo, rangeHi = lo, hi
	case strings.Contains(item, "-"):
		parts := strings.SplitN(item, "-", 2)
		a, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, fmt.Errorf("invalid range start in %q", item)
		}
		b, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("invalid range end in %q", item)
		}
		if a > b {
			return 0, fmt.Errorf("inverted range in %q", item)
		}
		rangeLo, rangeHi = a, b
	default:
		v, err := strconv.Atoi(item)
		if err != nil {
			return 0, fmt.Errorf("invalid value %q", item)
		}
		// Bare value with /step expands to v..hi step; otherwise v..v.
		rangeLo = v
		if step > 1 {
			rangeHi = hi
		} else {
			rangeHi = v
		}
	}
	if rangeLo < lo || rangeHi > hi {
		return 0, fmt.Errorf("value %d-%d out of allowed range [%d,%d]", rangeLo, rangeHi, lo, hi)
	}
	var bits uint64
	for v := rangeLo; v <= rangeHi; v += step {
		bits |= 1 << uint(v)
	}
	return bits, nil
}

func bit(mask uint64, i int) bool {
	if i < 0 || i >= 64 {
		return false
	}
	return mask&(1<<uint(i)) != 0
}
