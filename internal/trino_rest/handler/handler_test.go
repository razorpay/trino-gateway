package handler

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/razorpay/trino-gateway/internal/config"
	"github.com/razorpay/trino-gateway/internal/trino_rest/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func MockRows(columns []model.Column, rowData []map[string]interface{}) *sql.Rows {
	colNames := make([]string, len(columns))
	for i, col := range columns {
		colNames[i] = col.Name
	}
	rows := sqlmock.NewRows(colNames)
	for _, row := range rowData {
		values := make([]driver.Value, len(columns))
		for i, col := range columns {
			values[i] = row[col.Name]
		}
		rows.AddRow(values...)
	}
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectQuery("SELECT *").WillReturnRows(rows)
	result, _ := db.Query("SELECT * FROM mock_table")
	return result
}

// MockTrinoClient mocks the TrinoClient for testing purposes.
type MockTrinoClient struct {
	mock.Mock
}

func (m *MockTrinoClient) Query(query string) (*sql.Rows, error) {
	args := m.Called(query)

	// Retrieve the first value returned by the mock.Called method and cast it to a
	// *sql.Rows pointer.
	rows := args.Get(0).(*sql.Rows)

	// Retrieve the second value returned by the mock.Called method and cast it to an
	// error.
	err := args.Error(1)
	return rows, err
}

type MockQueryProcessor struct {
	mock.Mock
}

func (m *MockQueryProcessor) QueryResult(rows *sql.Rows) ([]model.Column, []map[string]interface{}, error) {
	args := m.Called(rows)

	// Retrieve the first value returned by the mock.Called method and cast it
	// to a slice of model.Column objects.
	columns := args.Get(0).([]model.Column)

	// Retrieve the second value returned by the mock.Called method and cast it
	// to a slice of map[string]interface{} objects.
	dataRows := args.Get(1).([]map[string]interface{})

	// Retrieve the third value returned by the mock.Called method and cast it
	// to an error.
	err := args.Error(2)
	return columns, dataRows, err
}

func TestQueryHandler_InvalidJSONPayload(t *testing.T) {
	mockProcessor := new(MockQueryProcessor)
	handler := NewHandler(&config.Config{}, mockProcessor)

	// Create a new HTTP POST request with an invalid JSON payload
	query := "SELECT * FROM table"
	reqData := &model.ReqData{SQL: query}
	reqBody, err := json.Marshal(reqData)
	if err != nil {
		t.Fatalf("Failed to marshal request data: %v", err)
	}
	req := httptest.NewRequest("POST", "/v1/query", bytes.NewBuffer(reqBody))

	// Create a new recorder to capture the response
	w := httptest.NewRecorder()

	// Call the QueryHandler function with the request and capture the response
	handler.QueryHandler().ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()

	// Assert that the response status code is 400 (Bad Request)
	assert.Equal(t, http.StatusBadRequest, result.StatusCode)
	var resp model.RespData
	err = json.NewDecoder(result.Body).Decode(&resp)
	assert.NoError(t, err)
	// Assert that the response body equals the error message "Invalid request payload"
	assert.Equal(t, "Invalid request payload", resp.Error.Message)
}

// Successfully decodes a valid JSON request payload
func TestQueryHandler_SuccessfulDecode(t *testing.T) {
	mockTrinoClient := new(MockTrinoClient)
	mockQueryProcessor := new(MockQueryProcessor)
	mockCfg := &config.Config{TrinoRest: config.TrinoRest{MaxRecords: 100}}
	handler := NewHandler(mockCfg, mockQueryProcessor)

	query := "SELECT * FROM table"
	reqData := &model.ReqData{SQL: query}
	reqBody, err := json.Marshal(reqData)
	if err != nil {
		t.Fatalf("Failed to marshal request data: %v", err)
	}
	req := httptest.NewRequest("POST", "/v1/query", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()

	columns := []model.Column{
		{Name: "col1", Type: "VARCHAR"},
		{Name: "col2", Type: "VARCHAR"},
	}
	rowData := []map[string]interface{}{{"col1": "data1", "col2": "data2"}}
	mockRows := MockRows(columns, rowData)
	mockTrinoClient.On("Query", query).Return(mockRows, nil)
	mockQueryProcessor.On("QueryResult", mockRows).Return(columns, rowData, nil)

	handler.QueryHandler().ServeHTTP(w, req)

	result := w.Result()
	defer result.Body.Close()

	assert.Equal(t, http.StatusAccepted, result.StatusCode)
	var resp model.RespData
	err = json.NewDecoder(result.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "Success", resp.Status)
	assert.Equal(t, columns, resp.Columns)
	assert.Equal(t, rowData, resp.Data)
	assert.Nil(t, resp.Error)
}

func TestQueryHnadler_TrinoQueryError(t *testing.T) {
	mockClient := new(MockTrinoClient)
	mockProcessor := new(MockQueryProcessor)
	handler := NewHandler(&config.Config{}, mockProcessor)

	// Set up the mock client so that it will return an error when
	// called with the query.
	query := "SELECT * FROM test limit 5"
	mockClient.On("Query", query).Return((*sql.Rows)(nil), errors.New("trino error"))

	// Marshal the request data (which is just the query string) into
	// a JSON byte slice.
	reqData := &model.ReqData{SQL: query}
	reqBody, err := json.Marshal(reqData)
	assert.NoError(t, err)

	// Create a new request with the query as the body.
	req, err := http.NewRequest("POST", "/v1/query", bytes.NewBuffer(reqBody))
	assert.NoError(t, err)

	// Create a new recorder which will be used to capture the
	// response.
	w := httptest.NewRecorder()

	// Call the handler with the request, and capture the response.
	handler.QueryHandler().ServeHTTP(w, req)

	result := w.Result()
	defer result.Body.Close()

	// Assert that the response was a 500 error.
	assert.Equal(t, http.StatusInternalServerError, result.StatusCode)
	var resp model.RespData
	err = json.NewDecoder(result.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "Error", resp.Status)
	assert.Equal(t, "trino error", resp.Error.Message)
}

func TestQueryHandler_QueryResultProcessingError(t *testing.T) {
	mockClient := new(MockTrinoClient)
	mockProcessor := new(MockQueryProcessor)
	handler := NewHandler(&config.Config{}, mockProcessor)

	// Define a query string that will be used to test the handler.
	query := "SELECT * FROM table"
	columns := []model.Column{
		{Name: "col1", Type: "VARCHAR"},
		{Name: "col2", Type: "VARCHAR"},
	}
	rowData := []map[string]interface{}{
		{"col1": "value1", "col2": "value2"},
	}
	mockRows := MockRows(columns, rowData)
	// Set up the mock TrinoClient so that when it is called with
	// the query string, it will return the mock sql.Rows object and
	// a nil error.
	mockClient.On("Query", query).Return(mockRows, nil)
	mockProcessor.On("QueryResult", mockRows).Return(columns, rowData, errors.New("processing error"))

	// Create a model.ReqData object that contains the query string.
	reqData := &model.ReqData{SQL: query}

	// Marshal the model.ReqData object into a JSON byte slice.
	reqBody, err := json.Marshal(reqData)
	if err != nil {
		t.Fatalf("Failed to marshal request data: %v", err)
	}

	req, err := http.NewRequest("POST", "/v1/query", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	w := httptest.NewRecorder()

	// Call the handler with the request, and capture the response.
	handler.QueryHandler().ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()

	// Assert that the response status code is 422 (Unprocessable Entity).
	assert.Equal(t, http.StatusUnprocessableEntity, result.StatusCode)

	var resp model.RespData
	err = json.NewDecoder(result.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "Unable to process: processing error", resp.Error.Message)
}

func TestQueryHandler_ResponseDataExceedMaxRecords(t *testing.T) {
	mockClient := new(MockTrinoClient)
	cfg := &config.Config{}
	cfg.TrinoRest.MaxRecords = 5
	mockProcessor := new(MockQueryProcessor)

	handler := NewHandler(cfg, mockProcessor)
	query := "SELECT * FROM tpch.sf1.nation limit 10"

	columns := []model.Column{
		{Name: "col1", Type: "VARCHAR"},
	}
	rowData := []map[string]interface{}{
		{"col1": "data1"},
		{"col1": "data2"},
		{"col1": "data3"},
		{"col1": "data4"},
		{"col1": "data5"},
		{"col1": "data6"},
	}
	mockRows := MockRows(columns, rowData)

	// Configure the mock TrinoClient to return a nil sql.Rows object and
	// a nil error when it is called with the query string.
	mockClient.On("Query", query).Return(mockRows, nil)

	// Configure the mock QueryProcessor to return a slice of model.Column
	// objects, a slice of map[string]interface{} objects, and a nil error
	// when it is called with a nil sql.Rows object.
	mockProcessor.On("QueryResult", mockRows).Return(columns, rowData, nil)

	// Create a model.ReqData object that contains the query string.
	reqData := &model.ReqData{SQL: query}
	reqBody, err := json.Marshal(reqData)
	if err != nil {
		t.Fatalf("Failed to marshal request data: %v", err)
	}

	req, err := http.NewRequest("POST", "/v1/query", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	w := httptest.NewRecorder()

	handler.QueryHandler().ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()

	// Assert that the response status code is 500 (Internal Server Error).
	assert.Equal(t, http.StatusInternalServerError, result.StatusCode)
	var resp model.RespData
	err = json.NewDecoder(result.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.Error.ErrorCode)
	assert.Equal(t, "exceeded allowed maximum number of records", resp.Error.Message)
}

func TestQueryHandler_QuerySuccess(t *testing.T) {
	mockClient := new(MockTrinoClient)
	cfg := &config.Config{}
	cfg.TrinoRest.MaxRecords = 10
	mockProcessor := new(MockQueryProcessor)
	handler := NewHandler(cfg, mockProcessor)

	query := "SELECT count(*) FROM tpch.sf1.nation"

	columns := []model.Column{
		{Name: "_col0", Type: "BIGINT"},
	}
	data := []map[string]interface{}{
		{"_col0": float64(25)},
	}
	mockRows := MockRows(columns, data)

	reqData := &model.ReqData{SQL: query}
	reqBody, err := json.Marshal(reqData)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "/v1/query", bytes.NewBuffer(reqBody))
	assert.NoError(t, err)

	w := httptest.NewRecorder()

	// Configure the mock TrinoClient to return a nil sql.Rows object and
	// a nil error when it is called with the query string.
	mockClient.On("Query", query).Return(mockRows, nil)

	// Configure the mock QueryProcessor to return the expected response
	// data when it is called with a nil sql.Rows object.
	mockProcessor.On("QueryResult", mockRows).Return(columns, data, nil)

	handler.QueryHandler().ServeHTTP(w, req)
	result := w.Result()
	defer result.Body.Close()

	// Assert that the response status code is 202 (Accepted).
	assert.Equal(t, http.StatusAccepted, result.StatusCode)

	var respData model.RespData
	err = json.NewDecoder(result.Body).Decode(&respData)
	assert.NoError(t, err)

	// Assert that the response data matches the expected response data.
	assert.Equal(t, "Success", respData.Status)
	assert.Equal(t, columns, respData.Columns)
	assert.Equal(t, data, respData.Data)
	assert.Nil(t, respData.Error)
}
