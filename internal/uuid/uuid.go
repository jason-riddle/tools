// Package uuid provides support for generating and manipulating UUIDs.
//
// See [RFC 9562] for details.
//
// Random components of new UUIDs are generated with a
// cryptographically secure random number generator.
//
// UUIDs may be generated using various algorithms.
// The [New] function returns a new UUID generated using
// an algorithm suitable for most purposes.
//
// [RFC 9562]: https://www.rfc-editor.org/rfc/rfc9562.html
package uuid

import (
	"cmp"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// UUID is a Universally Unique Identifier as specified in RFC 9562.
//
// UUIDs are comparable, such as with the == operator.
type UUID [16]byte

// Parse returns the UUID represented by s.
//
// It accepts strings in the following forms:
//
//	f81d4fae-7dec-11d0-a765-00a0c91e6bf6
//	{f81d4fae-7dec-11d0-a765-00a0c91e6bf6}
//	urn:uuid:f81d4fae-7dec-11d0-a765-00a0c91e6bf6
//	f81d4fae7dec11d0a76500a0c91e6bf6
//
// Alphabetic characters in the input may be any case.
func Parse(s string) (UUID, error) {
	switch {
	case len(s) == 36:
		// xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
		return parseDashed(s)
	case len(s) == 38 && s[0] == '{' && s[37] == '}':
		// {xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx}
		return parseDashed(s[1:37])
	case len(s) == 45 && strings.EqualFold(s[:9], "urn:uuid:"):
		// urn:uuid:xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
		return parseDashed(s[9:])
	case len(s) == 32:
		// xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
		return parseHex(s)
	default:
		return UUID{}, fmt.Errorf("uuid: invalid UUID %q", s)
	}
}

func parseDashed(s string) (UUID, error) {
	// Expect form: 8-4-4-4-12
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return UUID{}, fmt.Errorf("uuid: invalid UUID format %q", s)
	}
	// Remove dashes and parse as 32-char hex.
	plain := s[0:8] + s[9:13] + s[14:18] + s[19:23] + s[24:36]
	return parseHex(plain)
}

func parseHex(s string) (UUID, error) {
	if len(s) != 32 {
		return UUID{}, fmt.Errorf("uuid: invalid UUID hex length %d", len(s))
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return UUID{}, fmt.Errorf("uuid: invalid UUID hex %q: %w", s, err)
	}
	var u UUID
	copy(u[:], b)
	return u, nil
}

// MustParse returns the UUID represented by s.
//
// It panics if s is not a valid string representation of a UUID as defined by [Parse].
func MustParse(s string) UUID {
	u, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

// New returns a new UUID.
//
// Programs which do not have a need for a specific UUID generation algorithm
// should use New. At this time, New is equivalent to [NewV4].
func New() UUID {
	return NewV4()
}

// Nil returns the Nil UUID 00000000-0000-0000-0000-000000000000.
func Nil() UUID {
	return UUID{}
}

// Max returns the Max UUID ffffffff-ffff-ffff-ffff-ffffffffffff.
func Max() UUID {
	return UUID{
		0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff,
	}
}

// String returns the string representation of u.
//
// It uses the lowercase hex-and-dash representation defined in RFC 9562.
func (u UUID) String() string {
	var buf [36]byte
	encodeHex(buf[:], u)
	return string(buf[:])
}

func encodeHex(dst []byte, u UUID) {
	const hx = "0123456789abcdef"
	// 8-4-4-4-12
	groups := [5]struct{ start, end int }{
		{0, 4}, {4, 6}, {6, 8}, {8, 10}, {10, 16},
	}
	pos := 0
	for i, g := range groups {
		if i > 0 {
			dst[pos] = '-'
			pos++
		}
		for _, b := range u[g.start:g.end] {
			dst[pos] = hx[b>>4]
			dst[pos+1] = hx[b&0x0f]
			pos += 2
		}
	}
}

// MarshalText implements the [encoding.TextMarshaler] interface.
// The encoding is the same as returned by [UUID.String].
func (u UUID) MarshalText() ([]byte, error) {
	var buf [36]byte
	encodeHex(buf[:], u)
	return buf[:], nil
}

// AppendText implements the [encoding.TextAppender] interface.
// The encoding is the same as returned by [UUID.String].
func (u UUID) AppendText(b []byte) ([]byte, error) {
	var buf [36]byte
	encodeHex(buf[:], u)
	return append(b, buf[:]...), nil
}

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
// The UUID is expected in a form accepted by [Parse].
func (u *UUID) UnmarshalText(b []byte) error {
	parsed, err := Parse(string(b))
	if err != nil {
		return err
	}
	*u = parsed
	return nil
}

// Compare compares the UUID u with v.
// If u is before v, it returns -1.
// If u is after v, it returns +1.
// If they are the same, it returns 0.
//
// See [RFC 9562 section 6.11] for details on UUID sorting.
//
// [RFC 9562 section 6.11]: https://www.rfc-editor.org/rfc/rfc9562#section-6.11
func (u UUID) Compare(v UUID) int {
	for i := range u {
		if c := cmp.Compare(u[i], v[i]); c != 0 {
			return c
		}
	}
	return 0
}

// Version returns the version number of the UUID (1–8), or 0 if not a standard
// RFC 9562 version.
func (u UUID) Version() int {
	return int(u[6] >> 4)
}

// Variant returns a string describing the UUID variant field.
func (u UUID) Variant() string {
	b := u[8]
	switch {
	case b>>7 == 0:
		return "NCS backward compatibility"
	case b>>6 == 0b10:
		return "RFC 9562"
	case b>>5 == 0b110:
		return "Microsoft backward compatibility"
	default:
		return "reserved"
	}
}

// NewV4 returns a new version 4 UUID.
//
// Version 4 UUIDs contain 122 bits of random data.
func NewV4() UUID {
	var u UUID
	if _, err := rand.Read(u[:]); err != nil {
		panic(fmt.Errorf("uuid: failed to read random bytes: %w", err))
	}
	u[6] = (u[6] & 0x0f) | 0x40 // version 4
	u[8] = (u[8] & 0x3f) | 0x80 // variant 10xx (RFC 9562)
	return u
}

// v7state guards monotonic generation for NewV7.
var v7state struct {
	mu     sync.Mutex
	lastMs int64
	seq    uint16 // 12-bit sub-ms sequence, stored in low 12 bits
}

// NewV7 returns a new version 7 UUID.
//
// Version 7 UUIDs contain a timestamp in the most significant 48 bits,
// and at least 62 bits of random data.
//
// NewV7 always returns UUIDs which sort in increasing order,
// except when the system clock moves backwards.
func NewV7() UUID {
	ms := time.Now().UnixMilli()

	v7state.mu.Lock()
	if ms <= v7state.lastMs {
		// Clock has not advanced: increment sub-ms sequence to maintain order.
		v7state.seq++
		ms = v7state.lastMs
	} else {
		v7state.seq = 0
		v7state.lastMs = ms
	}
	seq := v7state.seq
	v7state.mu.Unlock()

	var u UUID

	// Bytes 0–5: 48-bit big-endian Unix millisecond timestamp.
	binary.BigEndian.PutUint64(u[0:], uint64(ms)<<16)
	// Shift the 48 ms bits into bytes 0-5 only.
	// PutUint64 wrote 8 bytes; overwrite bytes 6-7 below.

	// Bytes 6–7: version (4 bits) + 12-bit sequence counter.
	u[6] = 0x70 | byte(seq>>8) // version 7 | seq[11:8]
	u[7] = byte(seq & 0xff)    // seq[7:0]

	// Bytes 8–15: variant bits + 62 bits of random data.
	var tail [8]byte
	if _, err := rand.Read(tail[:]); err != nil {
		panic(fmt.Errorf("uuid: failed to read random bytes: %w", err))
	}
	tail[0] = (tail[0] & 0x3f) | 0x80 // variant 10xx
	copy(u[8:], tail[:])

	return u
}

var errInvalidUUID = errors.New("uuid: invalid UUID")

// IsNil reports whether u is the nil UUID.
func (u UUID) IsNil() bool {
	return u == Nil()
}

// IsMax reports whether u is the max UUID.
func (u UUID) IsMax() bool {
	return u == Max()
}
