package client

import (
	"encoding/json"
	"time"
)

// Duration is a wrapper around time.Duration that marshals to milliseconds.
//
// It marshals as a floating-point number of milliseconds so that sub-millisecond
// phases (DNS, TCP, TLS on fast/local paths) are not truncated to zero. Whole
// millisecond values still encode without a decimal point (e.g. 150), so existing
// integer consumers keep working while fractional values (e.g. 0.42) are preserved.
type Duration time.Duration

// MarshalJSON implements json.Marshaler, emitting fractional milliseconds.
func (d Duration) MarshalJSON() ([]byte, error) {
	ms := float64(d) / float64(time.Millisecond)
	return json.Marshal(ms)
}

// Milliseconds returns the duration as milliseconds
func (d Duration) Milliseconds() int64 {
	return time.Duration(d).Milliseconds()
}

// Seconds returns the duration as seconds
func (d Duration) Seconds() float64 {
	return time.Duration(d).Seconds()
}

// String returns the duration as a string
func (d Duration) String() string {
	return time.Duration(d).String()
}
