package proxy

import (
	"bytes"
	"io"
	"log"
	"net/http"

	"google.golang.org/protobuf/proto"

	"github.com/user/otlp9s/internal/decoder"
	"github.com/user/otlp9s/internal/model"

	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	colmetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

// HTTPProxy handles OTLP/HTTP (protobuf) on the three standard endpoints.
// It reads the body once, forwards the raw bytes to the backend, and
// decodes a copy into events.
type HTTPProxy struct {
	forwardURL string // e.g. "http://localhost:4318"
	events     chan<- model.Event
	client     *http.Client
}

func NewHTTPProxy(forwardURL string, events chan<- model.Event) *HTTPProxy {
	return &HTTPProxy{
		forwardURL: forwardURL,
		events:     events,
		client:     &http.Client{},
	}
}

func (p *HTTPProxy) ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/traces", p.handleTraces)
	mux.HandleFunc("/v1/metrics", p.handleMetrics)
	mux.HandleFunc("/v1/logs", p.handleLogs)

	log.Printf("[http] listening on %s, forwarding to %s", addr, p.forwardURL)
	return http.ListenAndServe(addr, mux)
}

func (p *HTTPProxy) handleTraces(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}

	// Forward raw bytes.
	go p.forward("/v1/traces", body, r.Header.Get("Content-Type"))

	// Decode.
	var req coltracepb.ExportTraceServiceRequest
	if err := proto.Unmarshal(body, &req); err != nil {
		log.Printf("[http] decode traces: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	data := &tracepb.TracesData{ResourceSpans: req.ResourceSpans}
	for _, ev := range decoder.DecodeTraces(data) {
		p.events <- ev
	}

	w.WriteHeader(http.StatusOK)
}

func (p *HTTPProxy) handleMetrics(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}

	go p.forward("/v1/metrics", body, r.Header.Get("Content-Type"))

	var req colmetricspb.ExportMetricsServiceRequest
	if err := proto.Unmarshal(body, &req); err != nil {
		log.Printf("[http] decode metrics: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	data := &metricspb.MetricsData{ResourceMetrics: req.ResourceMetrics}
	for _, ev := range decoder.DecodeMetrics(data) {
		p.events <- ev
	}

	w.WriteHeader(http.StatusOK)
}

func (p *HTTPProxy) handleLogs(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}

	go p.forward("/v1/logs", body, r.Header.Get("Content-Type"))

	var req collogspb.ExportLogsServiceRequest
	if err := proto.Unmarshal(body, &req); err != nil {
		log.Printf("[http] decode logs: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	data := &logspb.LogsData{ResourceLogs: req.ResourceLogs}
	for _, ev := range decoder.DecodeLogs(data) {
		p.events <- ev
	}

	w.WriteHeader(http.StatusOK)
}

func (p *HTTPProxy) forward(path string, body []byte, contentType string) {
	url := p.forwardURL + path
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		log.Printf("[http] forward request: %v", err)
		return
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	} else {
		req.Header.Set("Content-Type", "application/x-protobuf")
	}

	resp, err := p.client.Do(req)
	if err != nil {
		log.Printf("[http] forward %s: %v", path, err)
		return
	}
	resp.Body.Close()
}
