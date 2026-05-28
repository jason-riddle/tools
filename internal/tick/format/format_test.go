package format

import (
	"encoding/json"
	"testing"
	"time"
)

func TestLocationDefaultUTC(t *testing.T) {
	loc, err := Location("")
	if err != nil {
		t.Fatalf("Location() unexpected error: %v", err)
	}
	if loc != time.UTC {
		t.Fatalf("Location() = %v, want UTC", loc)
	}
}

func TestLocationLoadsLocation(t *testing.T) {
	loc, err := Location("America/New_York")
	if err != nil {
		t.Fatalf("Location() unexpected error: %v", err)
	}
	if got := loc.String(); got != "America/New_York" {
		t.Fatalf("Location() = %q, want %q", got, "America/New_York")
	}
}

func TestFormatDefaultRFC3339(t *testing.T) {
	ts := time.Date(2026, time.May, 27, 15, 4, 5, 123456789, time.UTC)
	if got := Format(ts, Options{}); got != "2026-05-27T15:04:05Z" {
		t.Fatalf("Format() = %q, want %q", got, "2026-05-27T15:04:05Z")
	}
}

func TestFormatNano(t *testing.T) {
	ts := time.Date(2026, time.May, 27, 15, 4, 5, 123456789, time.UTC)
	if got := Format(ts, Options{Mode: ModeRFC3339Nano}); got != "2026-05-27T15:04:05.123456789Z" {
		t.Fatalf("Format() = %q, want %q", got, "2026-05-27T15:04:05.123456789Z")
	}
}

func TestFormatLayout(t *testing.T) {
	ts := time.Date(2026, time.May, 27, 15, 4, 5, 0, time.UTC)
	if got := Format(ts, Options{Mode: ModeLayout, Layout: "2006-01-02 15:04:05 MST"}); got != "2026-05-27 15:04:05 UTC" {
		t.Fatalf("Format() = %q, want %q", got, "2026-05-27 15:04:05 UTC")
	}
}

func TestFormatOffset(t *testing.T) {
	ts := time.Date(2026, time.May, 27, 15, 4, 5, 0, time.UTC)
	if got := Format(ts, Options{Offset: 30 * time.Minute}); got != "2026-05-27T15:34:05Z" {
		t.Fatalf("Format() = %q, want %q", got, "2026-05-27T15:34:05Z")
	}
}

func TestFormatJSON(t *testing.T) {
	ts := time.Date(2026, time.May, 27, 15, 4, 5, 123456789, time.UTC)
	got := Format(ts, Options{Mode: ModeJSON})

	var data map[string]any
	if err := json.Unmarshal([]byte(got), &data); err != nil {
		t.Fatalf("json.Unmarshal() unexpected error: %v", err)
	}

	if data["RFC3339"] != "2026-05-27T15:04:05Z" {
		t.Fatalf("RFC3339 = %v, want %q", data["RFC3339"], "2026-05-27T15:04:05Z")
	}
	if data["epoch"] != float64(ts.Unix()) {
		t.Fatalf("epoch = %v, want %v", data["epoch"], ts.Unix())
	}
	if _, ok := data["DateTime"]; !ok {
		t.Fatal("DateTime missing from JSON output")
	}
}
