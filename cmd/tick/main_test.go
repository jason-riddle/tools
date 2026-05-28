package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseOptionsOffsetBeforeFlags(t *testing.T) {
	opts, err := parseOptions([]string{"+24h", "--nano"})
	if err != nil {
		t.Fatalf("parseOptions() unexpected error: %v", err)
	}
	if !opts.nano {
		t.Fatal("parseOptions() did not set nano mode")
	}
	if !opts.hasOffset || opts.offset != 24*time.Hour {
		t.Fatalf("parseOptions() offset = %v, hasOffset = %v", opts.offset, opts.hasOffset)
	}
}

func TestParseOptionsOffsetAfterFlags(t *testing.T) {
	opts, err := parseOptions([]string{"--epoch", "-90m"})
	if err != nil {
		t.Fatalf("parseOptions() unexpected error: %v", err)
	}
	if !opts.epoch {
		t.Fatal("parseOptions() did not set epoch mode")
	}
	if !opts.hasOffset || opts.offset != -90*time.Minute {
		t.Fatalf("parseOptions() offset = %v, hasOffset = %v", opts.offset, opts.hasOffset)
	}
}

func TestParseOptionsRejectsMultipleModes(t *testing.T) {
	_, err := parseOptions([]string{"--nano", "--epoch"})
	if err == nil {
		t.Fatal("parseOptions() expected an error for multiple output modes")
	}
}

func TestParseOptionsRejectsMultipleOffsets(t *testing.T) {
	_, err := parseOptions([]string{"+1h", "-30m"})
	if err == nil {
		t.Fatal("parseOptions() expected an error for multiple offsets")
	}
}

func TestLocationFromEnvDefaultUTC(t *testing.T) {
	loc, err := locationFromEnv("", false)
	if err != nil {
		t.Fatalf("locationFromEnv() unexpected error: %v", err)
	}
	if loc != time.UTC {
		t.Fatalf("locationFromEnv() = %v, want UTC", loc)
	}
}

func TestLocationFromEnvLoadsLocation(t *testing.T) {
	loc, err := locationFromEnv("America/New_York", true)
	if err != nil {
		t.Fatalf("locationFromEnv() unexpected error: %v", err)
	}
	if got := loc.String(); got != "America/New_York" {
		t.Fatalf("locationFromEnv() = %q, want %q", got, "America/New_York")
	}
}

func TestFormatTimeDefaultRFC3339(t *testing.T) {
	ts := time.Date(2026, time.May, 27, 15, 4, 5, 123456789, time.UTC)
	got, err := formatTime(ts, options{})
	if err != nil {
		t.Fatalf("formatTime() unexpected error: %v", err)
	}
	if want := "2026-05-27T15:04:05Z"; got != want {
		t.Fatalf("formatTime() = %q, want %q", got, want)
	}
}

func TestFormatTimeJSON(t *testing.T) {
	ts := time.Date(2026, time.May, 27, 15, 4, 5, 123456789, time.UTC)
	got, err := formatTime(ts, options{json: true})
	if err != nil {
		t.Fatalf("formatTime() unexpected error: %v", err)
	}

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
