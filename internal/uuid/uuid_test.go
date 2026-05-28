package uuid

import (
	"encoding/json"
	"strings"
	"testing"
)

// knownStr is a well-known UUID string used for parse round-trip tests.
const knownStr = "f81d4fae-7dec-11d0-a765-00a0c91e6bf6"

// TestParseAllForms verifies that all four accepted input forms parse to the same UUID.
func TestParseAllForms(t *testing.T) {
	forms := []string{
		"f81d4fae-7dec-11d0-a765-00a0c91e6bf6",
		"{f81d4fae-7dec-11d0-a765-00a0c91e6bf6}",
		"urn:uuid:f81d4fae-7dec-11d0-a765-00a0c91e6bf6",
		"f81d4fae7dec11d0a76500a0c91e6bf6",
	}
	want, err := Parse(forms[0])
	if err != nil {
		t.Fatalf("Parse(%q) unexpected error: %v", forms[0], err)
	}
	for _, f := range forms[1:] {
		got, err := Parse(f)
		if err != nil {
			t.Errorf("Parse(%q) unexpected error: %v", f, err)
			continue
		}
		if got != want {
			t.Errorf("Parse(%q) = %v, want %v", f, got, want)
		}
	}
}

// TestParseCaseInsensitive checks that uppercase hex is accepted.
func TestParseCaseInsensitive(t *testing.T) {
	upper := strings.ToUpper(knownStr)
	u, err := Parse(upper)
	if err != nil {
		t.Fatalf("Parse(%q) unexpected error: %v", upper, err)
	}
	if u.String() != knownStr {
		t.Errorf("String() = %q, want %q", u.String(), knownStr)
	}
}

// TestParseInvalidInputs verifies that malformed strings return errors.
func TestParseInvalidInputs(t *testing.T) {
	cases := []string{
		"",
		"not-a-uuid",
		"f81d4fae-7dec-11d0-a765-00a0c91e6bf",   // too short (35)
		"f81d4fae-7dec-11d0-a765-00a0c91e6bf67", // too long (37)
		"f81d4fae7dec11d0a76500a0c91e6bf",       // 31 chars
		"f81d4fae7dec11d0a76500a0c91e6bf67",     // 33 chars
		"f81d4faX-7dec-11d0-a765-00a0c91e6bf6",  // invalid hex
		"f81d4fae_7dec_11d0_a765_00a0c91e6bf6",  // underscores instead of dashes
	}
	for _, s := range cases {
		_, err := Parse(s)
		if err == nil {
			t.Errorf("Parse(%q) expected error, got nil", s)
		}
	}
}

// TestRoundTrip verifies that String -> Parse -> String is stable.
func TestRoundTrip(t *testing.T) {
	u := MustParse(knownStr)
	if got := u.String(); got != knownStr {
		t.Errorf("String() = %q, want %q", got, knownStr)
	}
}

// TestNil verifies the Nil UUID value and IsNil helper.
func TestNil(t *testing.T) {
	n := Nil()
	if n.String() != "00000000-0000-0000-0000-000000000000" {
		t.Errorf("Nil().String() = %q", n.String())
	}
	if !n.IsNil() {
		t.Error("IsNil() returned false for nil UUID")
	}
	if NewV4().IsNil() {
		t.Error("IsNil() returned true for random V4 UUID")
	}
}

// TestMax verifies the Max UUID value and IsMax helper.
func TestMax(t *testing.T) {
	m := Max()
	if m.String() != "ffffffff-ffff-ffff-ffff-ffffffffffff" {
		t.Errorf("Max().String() = %q", m.String())
	}
	if !m.IsMax() {
		t.Error("IsMax() returned false for max UUID")
	}
	if NewV4().IsMax() {
		t.Error("IsMax() returned true for random V4 UUID")
	}
}

// TestNewV4VersionAndVariant checks that version 4 and RFC 9562 variant bits are set.
func TestNewV4VersionAndVariant(t *testing.T) {
	for i := 0; i < 100; i++ {
		u := NewV4()
		if v := u.Version(); v != 4 {
			t.Errorf("NewV4().Version() = %d, want 4", v)
		}
		// Variant bits: top two bits of byte 8 must be 10.
		if u[8]>>6 != 0b10 {
			t.Errorf("NewV4() variant bits = %02b, want 10", u[8]>>6)
		}
	}
}

// TestNewV7VersionAndVariant checks that version 7 and RFC 9562 variant bits are set.
func TestNewV7VersionAndVariant(t *testing.T) {
	for i := 0; i < 100; i++ {
		u := NewV7()
		if v := u.Version(); v != 7 {
			t.Errorf("NewV7().Version() = %d, want 7", v)
		}
		if u[8]>>6 != 0b10 {
			t.Errorf("NewV7() variant bits = %02b, want 10", u[8]>>6)
		}
	}
}

// TestNewV7Monotonic verifies that rapidly generated V7 UUIDs are monotonically increasing.
func TestNewV7Monotonic(t *testing.T) {
	const n = 1000
	uuids := make([]UUID, n)
	for i := range uuids {
		uuids[i] = NewV7()
	}
	for i := 1; i < n; i++ {
		if uuids[i].Compare(uuids[i-1]) <= 0 {
			t.Errorf("NewV7 not monotonic at index %d: %v <= %v", i, uuids[i], uuids[i-1])
		}
	}
}

// TestCompare checks ordering semantics.
func TestCompare(t *testing.T) {
	a := MustParse("00000000-0000-0000-0000-000000000001")
	b := MustParse("00000000-0000-0000-0000-000000000002")
	if a.Compare(b) >= 0 {
		t.Errorf("expected a < b")
	}
	if b.Compare(a) <= 0 {
		t.Errorf("expected b > a")
	}
	if a.Compare(a) != 0 {
		t.Errorf("expected a == a")
	}
}

// TestMarshalUnmarshalText checks TextMarshaler/TextUnmarshaler round-trip.
func TestMarshalUnmarshalText(t *testing.T) {
	u := MustParse(knownStr)

	b, err := u.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() error: %v", err)
	}
	if string(b) != knownStr {
		t.Errorf("MarshalText() = %q, want %q", b, knownStr)
	}

	var u2 UUID
	if err := u2.UnmarshalText(b); err != nil {
		t.Fatalf("UnmarshalText() error: %v", err)
	}
	if u2 != u {
		t.Errorf("UnmarshalText() = %v, want %v", u2, u)
	}
}

// TestAppendText verifies AppendText appends correctly.
func TestAppendText(t *testing.T) {
	u := MustParse(knownStr)
	prefix := []byte("prefix:")
	got, err := u.AppendText(prefix)
	if err != nil {
		t.Fatalf("AppendText() error: %v", err)
	}
	want := "prefix:" + knownStr
	if string(got) != want {
		t.Errorf("AppendText() = %q, want %q", got, want)
	}
}

// TestJSONMarshal verifies that UUID marshals/unmarshals correctly in JSON.
func TestJSONMarshal(t *testing.T) {
	u := MustParse(knownStr)

	data, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}
	want := `"` + knownStr + `"`
	if string(data) != want {
		t.Errorf("json.Marshal() = %s, want %s", data, want)
	}

	var u2 UUID
	if err := json.Unmarshal(data, &u2); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if u2 != u {
		t.Errorf("json.Unmarshal() = %v, want %v", u2, u)
	}
}

// TestMustParsePanics ensures MustParse panics on invalid input.
func TestMustParsePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParse did not panic on invalid input")
		}
	}()
	MustParse("not-valid")
}

// TestVersion checks Version() for known UUIDs.
func TestVersion(t *testing.T) {
	// knownStr is a version-1 UUID (byte 6 high nibble = 1).
	u := MustParse(knownStr)
	if v := u.Version(); v != 1 {
		t.Errorf("Version() = %d, want 1 for %q", v, knownStr)
	}
	if v := NewV4().Version(); v != 4 {
		t.Errorf("NewV4().Version() = %d, want 4", v)
	}
	if v := NewV7().Version(); v != 7 {
		t.Errorf("NewV7().Version() = %d, want 7", v)
	}
}

// TestVariant checks Variant() for known UUIDs.
func TestVariant(t *testing.T) {
	u := NewV4()
	if v := u.Variant(); v != "RFC 9562" {
		t.Errorf("Variant() = %q, want %q", v, "RFC 9562")
	}
}

func TestNewWithOptions(t *testing.T) {
	u4, err := NewWithOptions(NewOptions{Version: 4})
	if err != nil {
		t.Fatalf("NewWithOptions(v4) unexpected error: %v", err)
	}
	if got := u4.Version(); got != 4 {
		t.Fatalf("NewWithOptions(v4) version = %d, want 4", got)
	}

	u7, err := NewWithOptions(NewOptions{Version: 7})
	if err != nil {
		t.Fatalf("NewWithOptions(v7) unexpected error: %v", err)
	}
	if got := u7.Version(); got != 7 {
		t.Fatalf("NewWithOptions(v7) version = %d, want 7", got)
	}

	_, err = NewWithOptions(NewOptions{Version: 9})
	if err == nil {
		t.Fatal("NewWithOptions(invalid) expected error")
	}
}

func TestDetails(t *testing.T) {
	u := MustParse(knownStr)
	d := u.Details()
	if d.UUID != knownStr {
		t.Fatalf("Details().UUID = %q, want %q", d.UUID, knownStr)
	}
	if d.Version != 1 {
		t.Fatalf("Details().Version = %d, want 1", d.Version)
	}
	if d.Variant != "RFC 9562" {
		t.Fatalf("Details().Variant = %q, want %q", d.Variant, "RFC 9562")
	}
	if d.Nil {
		t.Fatal("Details().Nil = true, want false")
	}
	if d.Max {
		t.Fatal("Details().Max = true, want false")
	}
}
