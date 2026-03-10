package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/user/otlp9s/internal/pipeline"
	"github.com/user/otlp9s/internal/proxy"
	"github.com/user/otlp9s/internal/store"
	"github.com/user/otlp9s/internal/tui"
)

func main() {
	var (
		listenAddr  string
		forwardAddr string
		enableGRPC  bool
		enableHTTP  bool
		bufferSize  int
		httpListen  string
	)

	root := &cobra.Command{
		Use:   "otlp9s",
		Short: "Interactive OTLP debugger — inspect OpenTelemetry traffic in your terminal",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default: enable gRPC if neither flag is set.
			if !enableGRPC && !enableHTTP {
				enableGRPC = true
			}

			if forwardAddr == "" {
				return fmt.Errorf("--forward is required (e.g. localhost:4317)")
			}

			// Suppress standard log output — it would corrupt the TUI.
			log.SetOutput(os.Stderr)

			// Core pipeline.
			buf := store.NewRingBuffer(bufferSize)
			idx := store.NewTraceIndex()
			router := pipeline.NewRouter(buf, idx)
			go router.Run()

			// Start proxy servers.
			if enableGRPC {
				grpcProxy := proxy.NewGRPCProxy(forwardAddr, router.Events)
				go func() {
					if err := grpcProxy.ListenAndServe(listenAddr); err != nil {
						log.Fatalf("gRPC server: %v", err)
					}
				}()
			}

			if enableHTTP {
				httpAddr := httpListen
				if httpAddr == "" {
					httpAddr = ":4318"
				}
				httpForward := "http://" + forwardAddr
				httpProxy := proxy.NewHTTPProxy(httpForward, router.Events)
				go func() {
					if err := httpProxy.ListenAndServe(httpAddr); err != nil {
						log.Fatalf("HTTP server: %v", err)
					}
				}()
			}

			// Start TUI.
			m := tui.NewModel(router)
			p := tea.NewProgram(m, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("TUI error: %w", err)
			}
			return nil
		},
	}

	flags := root.Flags()
	flags.StringVar(&listenAddr, "listen", ":4317", "gRPC listen address")
	flags.StringVar(&forwardAddr, "forward", "", "OTLP backend address to forward to (required)")
	flags.BoolVar(&enableGRPC, "grpc", false, "enable OTLP/gRPC proxy")
	flags.BoolVar(&enableHTTP, "http", false, "enable OTLP/HTTP proxy")
	flags.IntVar(&bufferSize, "buffer-size", 10000, "ring buffer capacity")
	flags.StringVar(&httpListen, "http-listen", ":4318", "HTTP listen address (when --http is set)")

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
