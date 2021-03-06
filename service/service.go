package service

import (
	"errors"
	"net/http"
	"os"

	"github.com/NYTimes/gizmo/config"
	"github.com/clawio/authentication/lib"
	"github.com/clawio/data/datacontroller"
	"github.com/clawio/data/datacontroller/simple"
	"github.com/clawio/sdk"
	"github.com/prometheus/client_golang/prometheus"
)

type (
	// Service implements server.Service and
	// handle all requests to the server.
	Service struct {
		Config         *Config
		SDK            *sdk.SDK
		DataController datacontroller.DataController
	}

	// Config is a struct that holds the
	// configuration for Service
	Config struct {
		Server         *config.Server
		General        *GeneralConfig
		DataController *DataControllerConfig
	}

	// GeneralConfig contains configuration parameters
	// for general parts of the service.
	GeneralConfig struct {
		BaseURL                      string
		JWTKey, JWTSigningMethod     string
		AuthenticationServiceBaseURL string
		RequestBodyMaxSize           int64
	}

	// DataControllerConfig is a struct that holds
	// configuration parameters for a data controller.
	DataControllerConfig struct {
		Type                       string
		SimpleDataDir              string
		SimpleTempDir              string
		SimpleChecksum             string
		SimpleVerifyClientChecksum bool
	}
)

// New will instantiate and return
// a new Service that implements server.Service.
func New(cfg *Config) (*Service, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}
	if cfg.General == nil {
		return nil, errors.New("config.General is nil")
	}
	if cfg.DataController == nil {
		return nil, errors.New("config.DataController is  nil")
	}

	urls := &sdk.ServiceEndpoints{}
	urls.AuthServiceBaseURL = cfg.General.AuthenticationServiceBaseURL
	s := sdk.New(urls, nil)

	dataController, err := getDataController(cfg.DataController)
	if err != nil {
		return nil, err
	}
	return &Service{Config: cfg, SDK: s, DataController: dataController}, nil
}

func getDataController(cfg *DataControllerConfig) (datacontroller.DataController, error) {
	opts := &simple.Options{
		DataDir:              cfg.SimpleDataDir,
		TempDir:              cfg.SimpleTempDir,
		Checksum:             cfg.SimpleChecksum,
		VerifyClientChecksum: cfg.SimpleVerifyClientChecksum,
	}
	// create DataDir and TempDir
	if err := os.MkdirAll(opts.DataDir, 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(opts.TempDir, 0755); err != nil {
		return nil, err
	}
	return simple.New(opts), nil
}

// Prefix returns the string prefix used for all endpoints within
// this service.
func (s *Service) Prefix() string {
	base := s.Config.General.BaseURL
	if base == "" {
		base = "/"
	}
	return base
}

// Middleware provides an http.Handler hook wrapped around all requests.
// In this implementation, we authenticate the request.
func (s *Service) Middleware(h http.Handler) http.Handler {
	return h
}

// Endpoints is a listing of all endpoints available in the Service.
func (s *Service) Endpoints() map[string]map[string]http.HandlerFunc {
	authenticator := lib.NewAuthenticator(s.Config.General.JWTKey, s.Config.General.JWTSigningMethod)
	return map[string]map[string]http.HandlerFunc{
		"/metrics": {
			"GET": func(w http.ResponseWriter, r *http.Request) {
				prometheus.Handler().ServeHTTP(w, r)
			},
		},
		"/upload/{path:.*}": {
			"PUT": prometheus.InstrumentHandlerFunc("/upload", authenticator.JWTHandlerFunc(s.Upload)),
		},
		"/download/{path:.*}": {
			"GET": prometheus.InstrumentHandlerFunc("/download", authenticator.JWTHandlerFunc(s.Download)),
		},
	}
}
