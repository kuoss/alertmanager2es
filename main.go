package main

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/kuoss/alertmanager2opensearch/config"
	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var (
	argparser *flags.Parser
	opts      config.Opts

	// Git version information
	gitCommit = "<unknown>"
	gitTag    = "<unknown>"
)

func main() {
	showHelp, err := initArgparser()
	if err != nil {
		log.Fatalf("Failed to init arg parser: %v", err)
	}
	if showHelp {
		argparser.WriteHelp(os.Stdout)
		os.Exit(0)
	}

	log.Infof("starting alertmanager2es v%s (%s; %s)", gitTag, gitCommit, runtime.Version())
	jsonString, err := opts.GetJson()
	if err != nil {
		log.Fatalf("Failed to get json: %v", err)
	}
	log.Info(jsonString)

	log.Infof("init exporter")
	exporter := &AlertmanagerOpenSearchExporter{}
	exporter.Init()

	cfg := opensearch.Config{
		Addresses: opts.OpenSearch.Addresses,
		Username:  opts.OpenSearch.Username,
		Password:  opts.OpenSearch.Password,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}
	if err := exporter.ConnectOpenSearch(cfg, opts.OpenSearch.Index); err != nil {
		log.Fatalf("failed to connect OpenSearch: %v", err)
	}

	// daemon mode
	log.Infof("starting http server on %s", opts.ServerBind)
	startHttpServer(exporter)
}

// init argparser and parse/validate arguments
func initArgparser() (showHelp bool, err error) {
	argparser = flags.NewParser(&opts, flags.Default)
	_, err = argparser.Parse()

	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			return true, nil
		}
		return false, fmt.Errorf("parse err: %w", err)
	}

	if opts.Logger.Verbose {
		log.SetLevel(log.DebugLevel)
	}
	if opts.Logger.Debug {
		log.SetReportCaller(true)
		log.SetLevel(log.TraceLevel)
		log.SetFormatter(&log.TextFormatter{
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				s := strings.Split(f.Function, ".")
				funcName := s[len(s)-1]
				return funcName, fmt.Sprintf("%s:%d", path.Base(f.File), f.Line)
			},
		})
	}
	if opts.Logger.LogJson {
		log.SetReportCaller(true)
		log.SetFormatter(&log.JSONFormatter{
			DisableTimestamp: true,
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				s := strings.Split(f.Function, ".")
				funcName := s[len(s)-1]
				return funcName, fmt.Sprintf("%s:%d", path.Base(f.File), f.Line)
			},
		})
	}

	return false, nil
}

// start and handle prometheus handler
func startHttpServer(exporter *AlertmanagerOpenSearchExporter) {
	// healthz
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprint(w, "Ok"); err != nil {
			log.Error(err)
		}
	})

	http.HandleFunc("/webhook", exporter.HttpHandler)
	http.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:         opts.ServerBind,
		ReadTimeout:  20 * time.Second,
		WriteTimeout: 20 * time.Second,
		IdleTimeout:  30 * time.Second,
	}
	log.Printf("Starting HTTP server on %s", opts.ServerBind)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
