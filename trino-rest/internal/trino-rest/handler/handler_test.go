package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"trino-api/internal/config"
	"trino-api/internal/model"
	"trino-api/internal/services/trino"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTrinoClient mocks the TrinoClient for testing purposes.
type MockTrinoClient struct {
	mock.Mock
	Db *trino.Client
}

func (m *MockTrinoClient) Query(query string) (*sql.Rows, error) {
	args := m.Called(query)
	return args.Get(0).(*sql.Rows), args.Error(1)
}

func TestInvalidPayload(t *testing.T) {
	mockClient := new(MockTrinoClient)
	cfg := &config.Config{}
	handler := NewHandler(mockClient.Db, cfg)

	req, err := http.NewRequest("POST", "/v1/query", bytes.NewBuffer([]byte("{invalid json")))
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	http.HandlerFunc(handler.QueryHandler()).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid request payload")
}

func TestTrinoQueryError(t *testing.T) {
	mockClient := new(MockTrinoClient)
	cfg := &config.Config{}
	handler := NewHandler(mockClient.Db, cfg)

	query := "SELECT * FROM test limit 5"
	mockClient.On("Query", query).Return(nil, errors.New("trino error"))

	reqData := &model.ReqData{SQL: query}
	reqBody, _ := json.Marshal(reqData)
	req, err := http.NewRequest("POST", "/v1/query", bytes.NewBuffer(reqBody))
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	http.HandlerFunc(handler.QueryHandler()).ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Unable to query trino: trino error")
}

func TestQueryResultError(t *testing.T) {
	mockClient := new(MockTrinoClient)
	cfg := &config.Config{}
	handler := NewHandler(mockClient.Db, cfg)

	rows := &sql.Rows{}
	query := "SELECT * FROM tpch.sf1.nation limit 5"
	mockClient.On("Query", query).Return(rows, nil)

	// queryResult := func(rows *sql.Rows) ([]string, [][]interface{}, error) {
	// 	return nil, nil, errors.New("processing error")
	// }

	reqData := &model.ReqData{SQL: query}
	reqBody, _ := json.Marshal(reqData)
	req, err := http.NewRequest("POST", "/v1/query", bytes.NewBuffer(reqBody))
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	http.HandlerFunc(handler.QueryHandler()).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "Unable to process: processing error")
}

func TestQueryResponseTooLarge(t *testing.T) {
	mockClient := new(MockTrinoClient)
	cfg := &config.Config{App: config.App{MaxRecords: 5}} // setting small limit for testing
	handler := NewHandler(mockClient.Db, cfg)

	rows := &sql.Rows{}
	query := "SELECT * FROM table"
	mockClient.On("Query", query).Return(rows, nil)

	// process.QueryResult = func(rows *sql.Rows) ([]string, [][]interface{}, error) {
	// 	return []string{"col1"}, [][]interface{}{
	// 		{"data1"}, {"data2"}, {"data3"}, {"data4"}, {"data5"}, {"data6"},
	// 	}, nil
	// }

	reqData := &model.ReqData{SQL: query}
	reqBody, _ := json.Marshal(reqData)
	req, err := http.NewRequest("POST", "/v1/query", bytes.NewBuffer(reqBody))
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	http.HandlerFunc(handler.QueryHandler()).ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	assert.Contains(t, w.Body.String(), "Response data is too big")
}

func TestQuerySuccess(t *testing.T) {
	mockClient := new(MockTrinoClient)
	mockRows := new(sql.Rows)
	mockClient.On("Query", "SELECT * FROM test").Return(mockRows, nil)

	cfg := &config.Config{
		App: config.App{MaxRecords: 10}}

	handler := NewHandler(mockClient.Db, cfg)

	rows := &sql.Rows{}
	mockClient.On("Query", "SELECT count(*) FROM tpch.sf1.nation").Return(rows, nil)

	reqData := &model.ReqData{SQL: "SELECT * FROM tpch.sf1.nation"}
	reqBody, _ := json.Marshal(reqData)
	req, err := http.NewRequest("POST", "/v1/query", bytes.NewBuffer(reqBody))
	assert.NoError(t, err)

	resp := httptest.NewRecorder()
	http.HandlerFunc(handler.QueryHandler()).ServeHTTP(resp, req)

	assert.Equal(t, http.StatusAccepted, resp.Code)

	var respData model.RespData
	err = json.Unmarshal(resp.Body.Bytes(), &respData)
	assert.NoError(t, err)

	assert.Equal(t, "Success", respData.Status)
	assert.Equal(t, []string{"_col0"}, respData.Columns)
	assert.Equal(t, [][]interface{}{{25}}, respData.Data)
	assert.Nil(t, respData.Error)
}

func TestHealthCheck(t *testing.T) {
	mockClient := new(MockTrinoClient)
	cfg := &config.Config{}
	handler := NewHandler(mockClient.Db, cfg)

	req, err := http.NewRequest("GET", "/v1/health", nil)
	assert.NoError(t, err)

	resp := httptest.NewRecorder()
	http.HandlerFunc(handler.HealthCheck()).ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var response map[string]string
	err = json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "OK", response["status"])
}
