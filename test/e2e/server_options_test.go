package e2e

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestServerTimeouts(t *testing.T) {
	t.Parallel()

	t.Run("read_timeout_closes_slow_clients", func(t *testing.T) {
		t.Parallel()

		mockLogstash := NewMockLogstashServer(t)
		exporter := StartExporter(t, NewConfigBuilder().
			AddInstance(mockLogstash.URL()).
			WithServerTimeouts(1, 0))

		conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", exporter.Port))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = conn.Close() }()

		// Send an incomplete request and never finish it
		if _, err := conn.Write([]byte("GET /metrics HTTP/1.1\r\nHost: localhost\r\n")); err != nil {
			t.Fatal(err)
		}

		start := time.Now()
		if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
			t.Fatal(err)
		}
		_, err = bufio.NewReader(conn).ReadByte()
		if err == nil {
			t.Fatal("expected the server to close the connection without a response")
		}
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			t.Fatal("expected the server to close the connection after the read timeout, but it stayed open")
		}
		if elapsed := time.Since(start); elapsed > 3*time.Second {
			t.Errorf("expected the connection to be closed after about 1s, took %s", elapsed)
		}
	})

	t.Run("write_timeout_aborts_slow_responses", func(t *testing.T) {
		t.Parallel()

		mockLogstash := NewMockLogstashServer(t)
		exporter := StartExporter(t, NewConfigBuilder().
			AddInstance(mockLogstash.URL()).
			WithHTTPTimeout("5s").
			WithServerTimeouts(0, 1))
		exporter.WaitForInstances(1)

		mockLogstash.SetDelay(2 * time.Second)

		_, err := exporter.Get("/metrics")
		if err == nil {
			t.Error("expected the response to be aborted after the write timeout")
		}
	})

	t.Run("fast_requests_are_not_affected", func(t *testing.T) {
		t.Parallel()

		mockLogstash := NewMockLogstashServer(t)
		exporter := StartExporter(t, NewConfigBuilder().
			AddInstance(mockLogstash.URL()).
			WithServerTimeouts(1, 1))

		exporter.WaitForInstances(1)
		resp, err := exporter.Get("/healthcheck")
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})
}
