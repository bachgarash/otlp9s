package proxy

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	_ "google.golang.org/grpc/encoding/gzip" // Register gzip decompressor

	"github.com/user/otlp9s/internal/decoder"
	"github.com/user/otlp9s/internal/model"

	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	colmetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

// GRPCProxy starts a gRPC server that implements the three OTLP collector
// services. Each service is a separate type to avoid Go method-name
// collisions (all three define an Export method).

type GRPCProxy struct {
	forwardAddr string
	events      chan<- model.Event
}

func NewGRPCProxy(forwardAddr string, events chan<- model.Event) *GRPCProxy {
	return &GRPCProxy{forwardAddr: forwardAddr, events: events}
}

func (p *GRPCProxy) ListenAndServe(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	srv := grpc.NewServer(grpc.MaxRecvMsgSize(16 * 1024 * 1024))
	coltracepb.RegisterTraceServiceServer(srv, &traceServer{p: p})
	colmetricspb.RegisterMetricsServiceServer(srv, &metricsServer{p: p})
	collogspb.RegisterLogsServiceServer(srv, &logsServer{p: p})

	log.Printf("[grpc] listening on %s, forwarding to %s", addr, p.forwardAddr)
	return srv.Serve(lis)
}

// dialBackend creates a connection to the forwarding target.
func (p *GRPCProxy) dialBackend() (*grpc.ClientConn, error) {
	return grpc.NewClient(p.forwardAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
}

// --- Trace Service ---

type traceServer struct {
	coltracepb.UnimplementedTraceServiceServer
	p *GRPCProxy
}

func (s *traceServer) Export(ctx context.Context, req *coltracepb.ExportTraceServiceRequest) (*coltracepb.ExportTraceServiceResponse, error) {
	// Forward in background — latency sensitive.
	go s.forward(req)

	data := &tracepb.TracesData{ResourceSpans: req.ResourceSpans}
	for _, ev := range decoder.DecodeTraces(data) {
		s.p.events <- ev
	}
	return &coltracepb.ExportTraceServiceResponse{}, nil
}

func (s *traceServer) forward(req *coltracepb.ExportTraceServiceRequest) {
	conn, err := s.p.dialBackend()
	if err != nil {
		log.Printf("[grpc] forward connect: %v", err)
		return
	}
	defer conn.Close()
	if _, err := coltracepb.NewTraceServiceClient(conn).Export(context.Background(), req); err != nil {
		log.Printf("[grpc] forward traces: %v", err)
	}
}

// --- Metrics Service ---

type metricsServer struct {
	colmetricspb.UnimplementedMetricsServiceServer
	p *GRPCProxy
}

func (s *metricsServer) Export(ctx context.Context, req *colmetricspb.ExportMetricsServiceRequest) (*colmetricspb.ExportMetricsServiceResponse, error) {
	// Tap only — metrics are already delivered to their real backend
	// by the collector. We just decode for display.
	data := &metricspb.MetricsData{ResourceMetrics: req.ResourceMetrics}
	for _, ev := range decoder.DecodeMetrics(data) {
		s.p.events <- ev
	}
	return &colmetricspb.ExportMetricsServiceResponse{}, nil
}

// --- Logs Service ---

type logsServer struct {
	collogspb.UnimplementedLogsServiceServer
	p *GRPCProxy
}

func (s *logsServer) Export(ctx context.Context, req *collogspb.ExportLogsServiceRequest) (*collogspb.ExportLogsServiceResponse, error) {
	// Tap only — logs are already delivered to their real backend
	// by the collector. We just decode for display.
	data := &logspb.LogsData{ResourceLogs: req.ResourceLogs}
	for _, ev := range decoder.DecodeLogs(data) {
		s.p.events <- ev
	}
	return &collogspb.ExportLogsServiceResponse{}, nil
}
