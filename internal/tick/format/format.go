// Package format renders time values in the output forms supported by tick.
package format

import (
	"encoding/json"
	"fmt"
	"time"
)

// Mode selects the output format for a time value.
type Mode int

const (
	ModeRFC3339 Mode = iota
	ModeRFC3339Nano
	ModeEpoch
	ModeLayout
	ModeJSON
)

// Options configures time formatting.
type Options struct {
	Mode   Mode
	Layout string
	Offset time.Duration
}

// Location returns the location named by tz, or UTC when tz is empty.
func Location(tz string) (*time.Location, error) {
	if tz == "" {
		return time.UTC, nil
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("load TZ location %q: %w", tz, err)
	}

	return loc, nil
}

// Format returns t rendered according to opts.
//
// Offset is applied before formatting.
func Format(t time.Time, opts Options) string {
	t = t.Add(opts.Offset)

	switch opts.Mode {
	case ModeRFC3339Nano:
		return t.Format(time.RFC3339Nano)
	case ModeEpoch:
		return fmt.Sprintf("%d", t.Unix())
	case ModeLayout:
		return t.Format(opts.Layout)
	case ModeJSON:
		return jsonTime(t)
	default:
		return t.Format(time.RFC3339)
	}
}

func jsonTime(t time.Time) string {
	data := map[string]any{
		"ANSIC":       t.Format(time.ANSIC),
		"DateOnly":    t.Format(time.DateOnly),
		"DateTime":    t.Format(time.DateTime),
		"Kitchen":     t.Format(time.Kitchen),
		"RFC822":      t.Format(time.RFC822),
		"RFC822Z":     t.Format(time.RFC822Z),
		"RFC850":      t.Format(time.RFC850),
		"RFC1123":     t.Format(time.RFC1123),
		"RFC1123Z":    t.Format(time.RFC1123Z),
		"RFC3339":     t.Format(time.RFC3339),
		"RFC3339Nano": t.Format(time.RFC3339Nano),
		"RubyDate":    t.Format(time.RubyDate),
		"Stamp":       t.Format(time.Stamp),
		"StampMicro":  t.Format(time.StampMicro),
		"StampMilli":  t.Format(time.StampMilli),
		"StampNano":   t.Format(time.StampNano),
		"TimeOnly":    t.Format(time.TimeOnly),
		"UnixDate":    t.Format(time.UnixDate),
		"epoch":       t.Unix(),
	}

	b, err := json.Marshal(data)
	if err != nil {
		panic(fmt.Errorf("marshal tick json output: %w", err))
	}

	return string(b)
}
