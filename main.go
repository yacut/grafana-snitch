package main

import (
	"flag"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

const defaultTimeout = 10

var (
	// DefaultHeaders sent always back to client
	DefaultHeaders = map[string]string{
		"Content-Type": "application/json",
	}
	promSuccess           *prometheus.CounterVec
	promErrors            *prometheus.CounterVec
	address               string
	config                Config
	configPath            string
	googleAdminConfigPath string
	googleAdminEmail      string
	grafanaHost           string
	grafanaUsername       string
	grafanaPassword       string
	updateInterval        time.Duration
	logLevel              string
	logJSON               bool
)

func main() {
	address = string(getEnv("LISTEN_ADDRESS", ":4949"))
	configPath = string(getEnv("CONFIG", ""))
	googleAdminConfigPath = string(getEnv("GOOGLE_APPLICATION_CREDENTIALS", ""))
	googleAdminEmail = string(getEnv("GOOGLE_ADMIN_EMAIL", ""))
	grafanaHost = string(getEnv("GRAFANA_HOST", ""))
	grafanaUsername = string(getEnv("GRAFANA_USERNAME", ""))
	grafanaPassword = string(getEnv("GRAFANA_PASSWORD", ""))
	updateIntervalEnv, err := strconv.ParseInt(getEnv("INTERVAL", "3600"), 10, 0)
	if err != nil {
		updateInterval = time.Duration(updateIntervalEnv) * time.Second
	}
	logJSONEnv, err := strconv.ParseBool(getEnv("LOG_JSON", "False"))
	if err != nil {
		logJSON = logJSONEnv
	}
	logLevel = string(getEnv("LOG_LEVEL", "info"))

	flag.StringVar(&address, "-listen-address", address, "The address to listen on for HTTP requests.")
	flag.StringVar(&configPath, "-config", configPath, "Path to yaml config")
	flag.StringVar(&googleAdminConfigPath, "-google-admin-config", googleAdminConfigPath, "The Path to the Service Account's Private Key file. see https://developers.google.com/admin-sdk/directory/v1/guides/delegation")
	flag.StringVar(&googleAdminEmail, "-google-admin-email", googleAdminEmail, "The Google Admin Email. see https://developers.google.com/admin-sdk/directory/v1/guides/delegation")
	flag.StringVar(&grafanaHost, "-grafana-host", grafanaHost, "Grafana server host")
	flag.StringVar(&grafanaUsername, "-grafana-username", grafanaUsername, "Grafana username")
	flag.StringVar(&grafanaPassword, "-grafana-password", grafanaPassword, "Grafana password")
	flag.DurationVar(&updateInterval, "-update-interval", updateInterval, "Update interval in seconds. e.g. 30s or 5m")
	flag.StringVar(&logLevel, "-log-level", logLevel, "Log level: `debug`, `info`, `warn`, `error`, `fatal` or `panic`.")
	flag.BoolVar(&logJSON, "-log-json", logJSON, "Log as JSON instead of the default ASCII formatter.")
	flag.Parse()

	logInit()
	prometheusInit()

	if configPath == "" {
		flag.Usage()
		log.Fatal().Msg("Missing --config")
	}
	if googleAdminConfigPath == "" {
		flag.Usage()
		log.Fatal().Msg("Missing --google-admin-config")
	}
	if googleAdminEmail == "" {
		flag.Usage()
		log.Fatal().Msg("Missing --google-admin-email")
	}
	if grafanaHost == "" {
		flag.Usage()
		log.Fatal().Msg("Missing --grafana-host")
	}
	if grafanaUsername == "" {
		flag.Usage()
		log.Fatal().Msg("Missing --grafana-username")
	}
	if grafanaPassword == "" {
		flag.Usage()
		log.Fatal().Msg("Missing --grafana-password")
	}

	configYaml, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Error reading YAML file")
	}
	err = yaml.Unmarshal(configYaml, &config)
	if err != nil {
		log.Fatal().Err(err).Msg("Error parsing YAML file")
	}
	log.Debug().Interface("config", config).Msg("Config loaded")

	router := mux.NewRouter()
	router.Use(loggingMiddleware)
	router.HandleFunc("/health", func(res http.ResponseWriter, req *http.Request) {
		respondwithJSON(res, http.StatusOK, map[string]bool{"ok": true})
	}).Methods(http.MethodGet, http.MethodOptions)
	router.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet, http.MethodOptions)
	router.NotFoundHandler = router.NewRoute().HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		respondwithJSON(res, http.StatusNotFound, map[string]string{"message": "not found"})
	}).GetHandler()

	srv := &http.Server{
		Handler:      router,
		Addr:         address,
		ReadTimeout:  defaultTimeout * time.Second,
		WriteTimeout: defaultTimeout * time.Second,
	}

	// Start Server
	go func() {
		log.Info().Str("address", address).Msg("Starting Server")
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal().Err(err)
		}
	}()

	go func() {
		for {
			// updateGoogleGroups()
			time.Sleep(updateInterval)
		}
	}()

	go func() {
		for {
			// checkGrafanaUsers()
			time.Sleep(updateInterval)
		}
	}()

	// Graceful Shutdown
	waitForShutdown(srv)
}
