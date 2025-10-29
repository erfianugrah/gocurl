package client

import (
	"crypto/tls"
	"net/http/httptrace"
	"time"
)

// TimingBreakdown contains detailed timing information for an HTTP request
type TimingBreakdown struct {
	DNSLookup        Duration `json:"dns_lookup"`
	TCPConnection    Duration `json:"tcp_connection"`
	TLSHandshake     Duration `json:"tls_handshake"`
	ServerProcessing Duration `json:"server_processing"`
	ContentTransfer  Duration `json:"content_transfer"`
	Total            Duration `json:"total"`

	ConnectionReused bool     `json:"connection_reused"`
	ConnectionIdle   bool     `json:"connection_idle"`
	IdleTime         Duration `json:"idle_time"`

	StatusCode       int               `json:"status_code"`
	ContentLength    int64             `json:"content_length"`
	ResponseSize     int64             `json:"response_size"`
	ResponseHeaders  map[string]string `json:"response_headers,omitempty"`
	ResponseBody     string            `json:"response_body,omitempty"`
	TLSVersion       string            `json:"tls_version,omitempty"`
	TLSCipherSuite   string            `json:"tls_cipher_suite,omitempty"`
	TLSServerName    string            `json:"tls_server_name,omitempty"`
	Error            string            `json:"error,omitempty"`

	// Streaming metrics (populated when --streaming flag is used)
	Streaming        *StreamMetrics    `json:"streaming,omitempty"`
}

// Tracer captures detailed timing information during HTTP request execution
type Tracer struct {
	dnsStart     time.Time
	dnsEnd       time.Time
	connStart    time.Time
	connEnd      time.Time
	tlsStart     time.Time
	tlsEnd       time.Time
	reqStart     time.Time
	respStart    time.Time
	respEnd      time.Time
	totalStart   time.Time

	tlsState     *tls.ConnectionState

	timing       *TimingBreakdown
}

// NewTracer creates a new Tracer instance
func NewTracer() *Tracer {
	return &Tracer{
		timing: &TimingBreakdown{},
	}
}

// ClientTrace returns an httptrace.ClientTrace configured to capture timing information
func (t *Tracer) ClientTrace() *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) {
			t.dnsStart = time.Now()
		},
		DNSDone: func(_ httptrace.DNSDoneInfo) {
			t.dnsEnd = time.Now()
		},
		ConnectStart: func(_, _ string) {
			t.connStart = time.Now()
		},
		ConnectDone: func(_, _ string, _ error) {
			t.connEnd = time.Now()
		},
		TLSHandshakeStart: func() {
			t.tlsStart = time.Now()
		},
		TLSHandshakeDone: func(state tls.ConnectionState, _ error) {
			t.tlsEnd = time.Now()
			t.tlsState = &state
		},
		WroteRequest: func(_ httptrace.WroteRequestInfo) {
			t.reqStart = time.Now()
		},
		GotFirstResponseByte: func() {
			t.respStart = time.Now()
		},
		GotConn: func(info httptrace.GotConnInfo) {
			t.timing.ConnectionReused = info.Reused
			t.timing.ConnectionIdle = info.WasIdle
			t.timing.IdleTime = Duration(info.IdleTime)
		},
	}
}

// Start marks the beginning of the overall request timing
func (t *Tracer) Start() {
	t.totalStart = time.Now()
}

// End marks the end of the overall request and calculates all durations
func (t *Tracer) End() {
	t.respEnd = time.Now()
	t.calculateDurations()
}

// calculateDurations computes all timing durations from captured timestamps
func (t *Tracer) calculateDurations() {
	// Calculate individual phase durations
	if !t.dnsStart.IsZero() && !t.dnsEnd.IsZero() {
		t.timing.DNSLookup = Duration(t.dnsEnd.Sub(t.dnsStart))
	}

	if !t.connStart.IsZero() && !t.connEnd.IsZero() {
		t.timing.TCPConnection = Duration(t.connEnd.Sub(t.connStart))
	}

	if !t.tlsStart.IsZero() && !t.tlsEnd.IsZero() {
		t.timing.TLSHandshake = Duration(t.tlsEnd.Sub(t.tlsStart))
	}

	if !t.reqStart.IsZero() && !t.respStart.IsZero() {
		t.timing.ServerProcessing = Duration(t.respStart.Sub(t.reqStart))
	}

	if !t.respStart.IsZero() && !t.respEnd.IsZero() {
		t.timing.ContentTransfer = Duration(t.respEnd.Sub(t.respStart))
	}

	if !t.totalStart.IsZero() {
		t.timing.Total = Duration(time.Since(t.totalStart))
	}

	// Populate TLS information if available
	if t.tlsState != nil {
		t.timing.TLSVersion = tlsVersionString(t.tlsState.Version)
		t.timing.TLSCipherSuite = tls.CipherSuiteName(t.tlsState.CipherSuite)
		t.timing.TLSServerName = t.tlsState.ServerName
	}
}

// tlsVersionString converts TLS version constant to string
func tlsVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return "Unknown"
	}
}

// Timing returns the captured timing breakdown
func (t *Tracer) Timing() *TimingBreakdown {
	return t.timing
}
