package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2/google"
	admin "google.golang.org/api/admin/directory/v1"
	"gopkg.in/yaml.v2"
)

const defaultTimeout = 10

// Config definition
type Config struct {
	Rules RuleConfigs `yaml:"rules"`
	Mode  string      `yaml:"mode"`
}

// RuleConfigs definition
type RuleConfigs struct {
	Groups []Rule `yaml:"groups"`
	Users  []Rule `yaml:"users"`
}

// Rule definition
type Rule struct {
	Name         string `yaml:"name"`
	Email        string `yaml:"email"`
	Organization string `yaml:"organization"`
	Role         string `yaml:"role"`
}

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

	// Graceful Shutdown
	waitForShutdown(srv)
}

func waitForShutdown(srv *http.Server) {
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive our signal.
	<-interruptChan

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	srv.Shutdown(ctx)

	log.Warn().Msg("Shutting down")
	os.Exit(0)
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func respondWithError(res http.ResponseWriter, code int, msg string) {
	respondwithJSON(res, code, map[string]string{"message": msg})
}

func respondwithJSON(res http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to marshal json")
	}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(code)
	res.Write(response)

}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		log.Debug().Str("url", req.RequestURI).Str("method", req.Method).Interface("headers", req.Header).Msg("Handle request")
		next.ServeHTTP(res, req)
	})
}

func logInit() {
	if logJSON == true {
		log.Logger = log.With().Caller().Logger()
	} else {
		log.Logger = log.With().Caller().Logger().Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
			NoColor:    false,
		})
	}
	switch logLevel {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case "panic":
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	}
	log.Debug().Msg("Logger initialized")
}

func prometheusInit() {
	promSuccess = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grafana_snitch_success",
			Help: "Cumulative number of role update operations",
		},
		[]string{"count"},
	)

	promErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grafana_snitch_errors",
			Help: "Cumulative number of errors during role update operations",
		},
		[]string{"count"},
	)
	prometheus.MustRegister(promSuccess)
	prometheus.MustRegister(promErrors)
}

// Build and returns an Admin SDK Directory service object authorized with
// the service accounts that act on behalf of the given user.
// Args:
//    googleAdminConfigPath: The Path to the Service Account's Private Key file
//    googleAdminEmail: The email of the user. Needs permissions to access the Admin APIs.
// Returns:
//    Admin SDK directory service object.
func getService(googleAdminConfigPath string, googleAdminEmail string) *admin.Service {
	jsonCredentials, err := ioutil.ReadFile(googleAdminConfigPath)
	if err != nil {
		promErrors.WithLabelValues("get-admin-config").Inc()
		log.Fatal().Err(err).Msg("Unable to read client secret file.")
		return nil
	}

	config, err := google.JWTConfigFromJSON(jsonCredentials, admin.AdminDirectoryGroupMemberReadonlyScope, admin.AdminDirectoryGroupReadonlyScope)
	if err != nil {
		promErrors.WithLabelValues("get-admin-config").Inc()
		log.Fatal().Err(err).Msg("Unable to parse client secret file to config.")
		return nil
	}
	config.Subject = googleAdminEmail
	ctx := context.Background()
	client := config.Client(ctx)

	srv, err := admin.New(client)
	if err != nil {
		promErrors.WithLabelValues("get-admin-client").Inc()
		log.Fatal().Err(err).Msg("Unable to retrieve Group Settings Client.")
		return nil
	}
	return srv
}

// Gets recursive the group members by email and returns the user list
// Args:
//    service: Admin SDK directory service object.
//    email: The email of the group.
// Returns:
//    Admin SDK member list.
func getMembers(service *admin.Service, email string) ([]*admin.Member, error) {
	result, err := service.Members.List(email).Do()
	if err != nil {
		promErrors.WithLabelValues("get-members").Inc()
		log.Fatal().Err(err).Msg("Unable to get group members.")
		return nil, err
	}

	var userList []*admin.Member
	for _, member := range result.Members {
		if member.Type == "GROUP" {
			groupMembers, _ := getMembers(service, member.Email)
			userList = append(userList, groupMembers...)
		} else {
			userList = append(userList, member)
		}
	}

	return userList, nil
}

// Remove duplicates from user list
// Args:
//    list: Admin SDK member list.
// Returns:
//    Admin SDK member list.
func uniq(list []*admin.Member) []*admin.Member {
	var uniqSet []*admin.Member
loop:
	for _, l := range list {
		for _, x := range uniqSet {
			if l.Email == x.Email {
				continue loop
			}
		}
		uniqSet = append(uniqSet, l)
	}

	return uniqSet
}
