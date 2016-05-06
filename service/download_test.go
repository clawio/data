package service

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/clawio/codes"
	"github.com/stretchr/testify/require"
)

// errorReader is a reader that always return an error
type errorReader struct{}

func (m *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("test error")
}

func (suite *TestSuite) TestDownload() {
	reader := strings.NewReader("1")
	suite.MockDataController.On("DownloadBLOB").Once().Return(reader, nil)
	r, err := http.NewRequest("GET", downloadURL+"myblob", nil)
	setToken(r)
	require.Nil(suite.T(), err)
	w := httptest.NewRecorder()
	suite.Server.ServeHTTP(w, r)
	require.Equal(suite.T(), http.StatusOK, w.Code)
	data, err := ioutil.ReadAll(w.Body)
	require.Nil(suite.T(), err)
	require.Equal(suite.T(), "1", string(data))
}
func (suite *TestSuite) TestDownload_withCodeNotFound() {
	reader := strings.NewReader("1")
	suite.MockDataController.On("DownloadBLOB").Once().Return(reader, codes.NewErr(codes.NotFound, ""))
	r, err := http.NewRequest("GET", downloadURL+"myblob", nil)
	setToken(r)
	require.Nil(suite.T(), err)
	require.Nil(suite.T(), err)
	w := httptest.NewRecorder()
	suite.Server.ServeHTTP(w, r)
	require.Equal(suite.T(), http.StatusNotFound, w.Code)
}
func (suite *TestSuite) TestDownload_withError() {
	reader := strings.NewReader("1")
	suite.MockDataController.On("DownloadBLOB").Once().Return(reader, errors.New("some error"))
	r, err := http.NewRequest("GET", downloadURL+"myblob", nil)
	setToken(r)
	require.Nil(suite.T(), err)
	require.Nil(suite.T(), err)
	w := httptest.NewRecorder()
	suite.Server.ServeHTTP(w, r)
	require.Equal(suite.T(), http.StatusInternalServerError, w.Code)
}
func (suite *TestSuite) TestDownload_withErrorCopying() {
	suite.MockDataController.On("DownloadBLOB").Once().Return(&errorReader{}, nil)
	r, err := http.NewRequest("GET", downloadURL+"myblob", nil)
	setToken(r)
	require.Nil(suite.T(), err)
	require.Nil(suite.T(), err)
	w := httptest.NewRecorder()
	suite.Server.ServeHTTP(w, r)
	require.Equal(suite.T(), http.StatusInternalServerError, w.Code)
}
