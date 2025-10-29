package client

import (
	"encoding/json"
	"time"
)

// Duration is a wrapper around time.Duration that marshals to milliseconds
type Duration time.Duration

// MarshalJSON implements json.Marshaler
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).Milliseconds())
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
