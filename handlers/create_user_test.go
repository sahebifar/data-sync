package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCreateUserHandlerRejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "invalid JSON",
			body: `{"first_name":`,
		},
		{
			name: "missing required fields",
			body: `{"first_name":"Amir"}`,
		},
		{
			name: "empty first name",
			body: `{
				"first_name":" ",
				"last_name":"Test",
				"email":"amir@test.com",
				"age":25
			}`,
		},
		{
			name: "invalid email",
			body: `{
				"first_name":"Amir",
				"last_name":"Test",
				"email":"not-an-email",
				"age":25
			}`,
		},
		{
			name: "age below minimum",
			body: `{
				"first_name":"Amir",
				"last_name":"Test",
				"email":"amir@test.com",
				"age":0
			}`,
		},
		{
			name: "age above maximum",
			body: `{
				"first_name":"Amir",
				"last_name":"Test",
				"email":"amir@test.com",
				"age":121
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/users/create",
				strings.NewReader(tt.body),
			)
			req.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()
			handler := CreateUserHandler(nil, nil)
			handler.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusBadRequest {
				t.Errorf(
					"expected status %d, got %d; response: %s",
					http.StatusBadRequest,
					recorder.Code,
					recorder.Body.String(),
				)
			}
		})
	}
}

func TestCreateUserHandlerRejectsNonPostMethod(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/users/create",
		nil,
	)

	recorder := httptest.NewRecorder()
	handler := CreateUserHandler(nil, nil)
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"expected status %d, got %d; response: %s",
			http.StatusMethodNotAllowed,
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

func TestCreateUserHandlerRejectsDuplicateEmail(t *testing.T) {
	postgresDB, pgMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create PostgreSQL mock: %v", err)
	}
	defer postgresDB.Close()

	yugabyteDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create YugabyteDB mock: %v", err)
	}
	defer yugabyteDB.Close()

	pgMock.ExpectQuery(`SELECT EXISTS`).
		WithArgs("amir@test.com").
		WillReturnRows(
			sqlmock.NewRows([]string{"exists"}).AddRow(true),
		)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/users/create",
		strings.NewReader(`{
			"first_name":"Amir",
			"last_name":"Test",
			"email":"amir@test.com",
			"age":25
		}`),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	handler := CreateUserHandler(postgresDB, yugabyteDB)
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusConflict {
		t.Fatalf(
			"expected status %d, got %d; response: %s",
			http.StatusConflict,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if err := pgMock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet PostgreSQL expectations: %v", err)
	}
}
