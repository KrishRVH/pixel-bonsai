package usage

import (
	"bytes"
	"strconv"
	"strings"
	"time"
)

// A count is a token count in a log: a nonnegative JSON integer. Anything else, such as a float, a string, a boolean,
// or a negative number, reads as 0, so one odd field does not drop the rest of a line.
type count int64

// UnmarshalJSON implements [encoding/json.Unmarshaler].
func (c *count) UnmarshalJSON(b []byte) error {
	n, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil || n < 0 {
		n = 0
	}
	*c = count(n)
	return nil
}

func sum(counts ...count) int64 {
	total := int64(0)
	for _, c := range counts {
		total += int64(c)
	}
	return total
}

// A stamp is an event time in a log: an RFC 3339 string with a time zone, or an integer of Unix milliseconds. Raw
// keeps the JSON text, which some sources use in an event's key. Anything else reads as the zero time, which the
// ledger rejects.
type stamp struct {
	Time time.Time
	Raw  string
}

// UnmarshalJSON implements [encoding/json.Unmarshaler].
func (s *stamp) UnmarshalJSON(b []byte) error {
	s.Raw = string(b)
	s.Time = parseStamp(b)
	return nil
}

func parseStamp(b []byte) time.Time {
	if ms, err := strconv.ParseInt(string(b), 10, 64); err == nil {
		return timeFromMillis(ms)
	}
	if len(b) < 2 || b[0] != '"' {
		return time.Time{}
	}
	t, err := parseTime(string(bytes.Trim(b, `"`)))
	if err != nil {
		return time.Time{}
	}
	return t
}

func timeFromMillis(ms int64) time.Time {
	return time.UnixMilli(ms).UTC()
}

// parseTime reads an RFC 3339 time with a time zone offset or Z, also with a space between the date and the time,
// and returns it in UTC.
func parseTime(s string) (time.Time, error) {
	if len(s) > 10 && s[10] == ' ' {
		s = s[:10] + "T" + s[11:]
	}
	t, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// assistant is the role of a model's replies in the harnesses' logs.
const assistant = "assistant"

// key joins the parts of an event's identity with a separator no part contains.
func key(parts ...string) string {
	return strings.Join(parts, "\x00")
}
