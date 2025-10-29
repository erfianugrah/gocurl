package client

import (
	"testing"
	"time"
)

func TestNewTracer(t *testing.T) {
	tracer := NewTracer()

	if tracer == nil {
		t.Fatal("NewTracer returned nil")
	}

	if tracer.timing == nil {
		t.Error("Tracer timing should be initialized")
	}
}

func TestTracerTiming(t *testing.T) {
	tracer := NewTracer()

	// Simulate timing events
	tracer.dnsStart = time.Now()
	time.Sleep(10 * time.Millisecond)
	tracer.dnsEnd = time.Now()

	tracer.connStart = time.Now()
	time.Sleep(20 * time.Millisecond)
	tracer.connEnd = time.Now()

	tracer.calculateDurations()

	timing := tracer.Timing()

	if timing.DNSLookup < Duration(10*time.Millisecond) {
		t.Errorf("DNS lookup duration too short: %v", timing.DNSLookup)
	}

	if timing.TCPConnection < Duration(20*time.Millisecond) {
		t.Errorf("TCP connection duration too short: %v", timing.TCPConnection)
	}
}

func TestTracerStartEnd(t *testing.T) {
	tracer := NewTracer()

	tracer.Start()
	time.Sleep(50 * time.Millisecond)
	tracer.End()

	timing := tracer.Timing()

	if timing.Total < Duration(50*time.Millisecond) {
		t.Errorf("Total duration too short: %v", timing.Total)
	}
}

func TestTimingBreakdown(t *testing.T) {
	timing := &TimingBreakdown{
		DNSLookup:        Duration(10 * time.Millisecond),
		TCPConnection:    Duration(20 * time.Millisecond),
		TLSHandshake:     Duration(30 * time.Millisecond),
		ServerProcessing: Duration(40 * time.Millisecond),
		ContentTransfer:  Duration(50 * time.Millisecond),
		Total:            Duration(150 * time.Millisecond),
		StatusCode:       200,
		ConnectionReused: false,
	}

	if timing.DNSLookup != Duration(10*time.Millisecond) {
		t.Errorf("Expected DNS lookup 10ms, got %v", timing.DNSLookup)
	}

	if timing.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", timing.StatusCode)
	}

	if timing.ConnectionReused {
		t.Error("Expected connection not reused")
	}
}

func TestTracerConnectionInfo(t *testing.T) {
	tracer := NewTracer()
	timing := tracer.Timing()

	// Initially should not be reused
	if timing.ConnectionReused {
		t.Error("New connection should not be marked as reused")
	}

	if timing.ConnectionIdle {
		t.Error("New connection should not be marked as idle")
	}

	if timing.IdleTime != 0 {
		t.Error("New connection should have zero idle time")
	}
}
