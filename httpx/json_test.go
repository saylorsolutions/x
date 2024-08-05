package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

type TestErrorType struct {
	Error string `json:"error"`
}

type TestRequestType struct {
	Word string `json:"data"`
}

type TestResponseType struct {
	Repeated string `json:"reversed"`
}

func TestHandleJSON(t *testing.T) {
	var (
		errorHappened  bool
		requestHandled bool
		serverError    bool
		clientError    bool
	)
	errHandler := JSONErrorHandler[TestErrorType](func(err error) TestErrorType {
		errorHappened = true
		if errors.Is(err, ErrServerError) {
			serverError = true
		}
		if errors.Is(err, ErrClientError) {
			clientError = true
		}
		return TestErrorType{
			Error: err.Error(),
		}
	})
	handler := HandleJSON(errHandler, func(body *TestRequestType) (*TestResponseType, error) {
		requestHandled = true
		if body.Word == "error" {
			return nil, errors.New("error")
		}
		return &TestResponseType{
			Repeated: body.Word,
		}, nil
	})

	t.Run("Happy path", func(t *testing.T) {
		serverError = false
		clientError = false
		errorHappened = false
		requestHandled = false
		recorder := httptest.NewRecorder()
		goodBody, err := json.Marshal(TestRequestType{Word: "test"})
		assert.NoError(t, err)
		req, err := http.NewRequest("GET", "/test", bytes.NewReader(goodBody))
		assert.NoError(t, err)
		handler.ServeHTTP(recorder, req)

		assert.False(t, serverError, "Should not be a server error")
		assert.False(t, clientError, "Should not be a client error")
		assert.False(t, errorHappened, "Unexpected error happened")
		assert.True(t, requestHandled, "Request should have been handled")
		assert.Equal(t, 200, recorder.Code)
		assert.Equal(t, ContentTypeJSON, recorder.Header().Get(HeaderContentType))
	})

	t.Run("Server error", func(t *testing.T) {
		serverError = false
		clientError = false
		errorHappened = false
		requestHandled = false
		recorder := httptest.NewRecorder()
		errBody, err := json.Marshal(TestRequestType{Word: "error"})
		assert.NoError(t, err)
		req, err := http.NewRequest("GET", "/test", bytes.NewReader(errBody))
		assert.NoError(t, err)
		handler.ServeHTTP(recorder, req)

		assert.True(t, serverError, "Should be a server error")
		assert.False(t, clientError, "Should not be a client error")
		assert.True(t, errorHappened, "An error should have been returned")
		assert.True(t, requestHandled, "Request should have been handled")
		assert.Equal(t, 500, recorder.Code)
		assert.Equal(t, ContentTypeJSON, recorder.Header().Get(HeaderContentType))
	})

	t.Run("Client error", func(t *testing.T) {
		serverError = false
		clientError = false
		errorHappened = false
		requestHandled = false
		recorder := httptest.NewRecorder()
		notJSON := []byte("this is a string, not JSON!")
		req, err := http.NewRequest("GET", "/test", bytes.NewReader(notJSON))
		assert.NoError(t, err)
		handler.ServeHTTP(recorder, req)

		assert.False(t, serverError, "Should not be a server error")
		assert.True(t, clientError, "Should be a client error")
		assert.True(t, errorHappened, "An error should have been returned")
		assert.False(t, requestHandled, "Request should have been caught by HandleJSON")
		assert.Equal(t, 400, recorder.Code)
		assert.Equal(t, ContentTypeJSON, recorder.Header().Get(HeaderContentType))
	})
}
