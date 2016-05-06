package service

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/NYTimes/gizmo/config"
	"github.com/NYTimes/gizmo/server"
	"github.com/clawio/authentication/lib"
	mock_datacontroller "github.com/clawio/data/datacontroller/mock"
	"github.com/clawio/entities"
	"github.com/clawio/sdk"
	"github.com/clawio/sdk/mocks"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	downloadURL string
	uploadURL   string
	metricsURL  string
	user        = &entities.User{Username: "test"}
	jwtToken    string
)

type TestSuite struct {
	suite.Suite
	MockDataController *mock_datacontroller.DataController
	SDK                *sdk.SDK
	Service            *Service
	Server             *server.SimpleServer
}

func Test(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (suite *TestSuite) SetupTest() {
	cfg := &Config{
		Server: &config.Server{},
		General: &GeneralConfig{
			JWTKey:             "secret",
			JWTSigningMethod:   "HS256",
			RequestBodyMaxSize: 1024, // 1KiB
		},
		DataController: &DataControllerConfig{
			Type:          "simple",
			SimpleDataDir: "/tmp",
			SimpleTempDir: "/tmp",
		},
	}
	mockAuthService := &mocks.MockAuthService{}
	s := &sdk.SDK{}
	s.Auth = mockAuthService

	svc := &Service{}
	svc.SDK = s
	svc.Config = cfg

	mockDataController := &mock_datacontroller.DataController{}
	svc.DataController = mockDataController
	suite.MockDataController = mockDataController

	suite.Service = svc
	serv := server.NewSimpleServer(cfg.Server)
	serv.Register(suite.Service)
	suite.Server = serv
	// create homedir for user test
	err := os.MkdirAll("/tmp/t/test", 0755)
	require.Nil(suite.T(), err)

	// Create the token
	authenticator := lib.NewAuthenticator(cfg.General.JWTKey, cfg.General.JWTSigningMethod)
	token, err := authenticator.CreateToken(user)
	require.Nil(suite.T(), err)
	jwtToken = token

	// set testing urls
	uploadURL = path.Join(svc.Config.General.BaseURL, "/upload") + "/"
	downloadURL = path.Join(svc.Config.General.BaseURL, "/download") + "/"
	metricsURL = path.Join(svc.Config.General.BaseURL, "/metrics")
}

func (suite *TestSuite) TeardownTest() {
	os.Remove("/tmp/t/test")
}

func (suite *TestSuite) TestNew() {
	cfg := &Config{
		Server: &config.Server{},
		General: &GeneralConfig{
			RequestBodyMaxSize: 1024, // 1KiB
		},
		DataController: &DataControllerConfig{
			Type:          "simple",
			SimpleDataDir: "/tmp",
			SimpleTempDir: "/tmp",
		},
	}
	svc, err := New(cfg)
	require.Nil(suite.T(), err)
	require.NotNil(suite.T(), svc)
}
func (suite *TestSuite) TestgetDataController_withBadDataDir() {
	cfg := &DataControllerConfig{
		SimpleDataDir: "/i/cannot/write/here",
	}
	_, err := getDataController(cfg)
	require.NotNil(suite.T(), err)
}
func (suite *TestSuite) TestgetDataController_withBadTempDir() {
	cfg := &DataControllerConfig{
		SimpleDataDir: "/tmp",
		SimpleTempDir: "/i/cannot/write/here",
	}
	_, err := getDataController(cfg)
	require.NotNil(suite.T(), err)
}
func (suite *TestSuite) TestNew_withNilConfig() {
	_, err := New(nil)
	require.NotNil(suite.T(), err)
}

func (suite *TestSuite) TestNew_withNilGeneralConfig() {
	cfg := &Config{
		Server:  nil,
		General: nil,
	}
	_, err := New(cfg)
	require.NotNil(suite.T(), err)
}
func (suite *TestSuite) TestNew_withNilDataControllerConfig() {
	cfg := &Config{
		Server:         nil,
		General:        &GeneralConfig{},
		DataController: nil,
	}
	_, err := New(cfg)
	require.NotNil(suite.T(), err)
}
func (suite *TestSuite) TestNew_withBadDataController() {
	cfg := &Config{
		Server:         nil,
		General:        &GeneralConfig{},
		DataController: &DataControllerConfig{SimpleDataDir: "/i/cannot/write/here"},
	}
	_, err := New(cfg)
	require.NotNil(suite.T(), err)
}
func (suite *TestSuite) TestPrefix() {
	if suite.Service.Config.General.BaseURL == "" {
		require.Equal(suite.T(), suite.Service.Prefix(), "/")
	} else {
		require.Equal(suite.T(), suite.Service.Config.General.BaseURL, suite.Service.Prefix())
	}
}

func (suite *TestSuite) TestMetrics() {
	r, err := http.NewRequest("GET", metricsURL, nil)
	require.Nil(suite.T(), err)
	w := httptest.NewRecorder()
	suite.Server.ServeHTTP(w, r)
	require.Equal(suite.T(), 200, w.Code)
}
func (suite *TestSuite) TestAuthenticateHandlerFunc() {
	r, err := http.NewRequest("PUT", uploadURL+"myblob", nil)
	require.Nil(suite.T(), err)
	setToken(r)
	w := httptest.NewRecorder()
	suite.Server.ServeHTTP(w, r)
	require.NotEqual(suite.T(), 401, w.Code)
}
func (suite *TestSuite) TestAuthenticateHandlerFunc_withBadToken() {
	r, err := http.NewRequest("PUT", uploadURL+"myblob", nil)
	require.Nil(suite.T(), err)
	r.Header.Set("Authorization", " Bearer fake")
	w := httptest.NewRecorder()
	suite.Server.ServeHTTP(w, r)
	require.Equal(suite.T(), 401, w.Code)
}

func setToken(r *http.Request) {
	r.Header.Set("Authorization", "bearer "+jwtToken)

}
