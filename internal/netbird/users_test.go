package netbird

import (
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestListUsersRequest(t *testing.T) {
	yes, no := true, false
	tests := []struct {
		name      string
		params    ListUsersParams
		wantQuery string
	}{
		{name: "nil service_user", wantQuery: ""},
		{name: "service_user true", params: ListUsersParams{ServiceUser: &yes}, wantQuery: "service_user=true"},
		{name: "service_user false", params: ListUsersParams{ServiceUser: &no}, wantQuery: "service_user=false"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got *http.Request
			c := newTestClient(t, "/api", func(w http.ResponseWriter, r *http.Request) {
				got = r
				_, _ = w.Write([]byte("[]"))
			})
			if _, err := c.ListUsers(t.Context(), tt.params); err != nil {
				t.Fatalf("ListUsers: %v", err)
			}
			if got.Method != http.MethodGet {
				t.Errorf("method = %s, want GET", got.Method)
			}
			if got.URL.Path != "/api/users" {
				t.Errorf("path = %q, want /api/users", got.URL.Path)
			}
			if got.URL.RawQuery != tt.wantQuery {
				t.Errorf("query = %q, want %q", got.URL.RawQuery, tt.wantQuery)
			}
			if h := got.Header.Get("Authorization"); h != "Token "+sentinelToken {
				t.Errorf("Authorization = %q", h)
			}
			if h := got.Header.Get("Accept"); h != "application/json" {
				t.Errorf("Accept = %q", h)
			}
		})
	}
}

func TestListUsersDecode(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []User
	}{
		{name: "null", body: "null", want: nil},
		{name: "empty", body: "[]", want: []User{}},
		{
			name: "full user with unknown fields",
			body: `[{"id":"google-oauth2|1","email":"a@example.com","name":"Ann","role":"admin","status":"active",
				"last_login":"2023-05-05T09:00:35.477782Z","auto_groups":["g1","g2"],"is_current":true,
				"is_service_user":false,"is_blocked":true,"pending_approval":true,"issued":"api","idp_id":"okta-abc123",
				"password":"hunter2","permissions":{"is_restricted":false,"modules":{"peers":{"read":true}}},"future_field":1}]`,
			want: []User{{
				ID: "google-oauth2|1", Email: "a@example.com", Name: "Ann", Role: "admin", Status: "active",
				LastLogin: "2023-05-05T09:00:35.477782Z", AutoGroups: []string{"g1", "g2"}, IsCurrent: true,
				IsServiceUser: false, IsBlocked: true, PendingApproval: true, Issued: "api", IdpID: "okta-abc123",
			}},
		},
		{
			name: "service user",
			body: `[{"id":"s1","email":"","name":"bot","role":"user","is_service_user":true}]`,
			want: []User{{ID: "s1", Name: "bot", Role: "user", IsServiceUser: true}},
		},
		{
			name: "optional fields absent",
			body: `[{"id":"u2","email":"b@example.com","name":"Bob","role":"user"}]`,
			want: []User{{ID: "u2", Email: "b@example.com", Name: "Bob", Role: "user"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newTestClient(t, "/api", func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tt.body))
			})
			got, err := c.ListUsers(t.Context(), ListUsersParams{})
			if err != nil {
				t.Fatalf("ListUsers: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestListUsersAPIError(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		body        string
		wantMessage string
	}{
		{name: "json message", status: 403, body: `{"message":"no permission"}`, wantMessage: "no permission"},
		{name: "token echoed", status: 400, body: `{"message":"bad token nbp_SENTINEL here"}`, wantMessage: "Bad Request"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := newTestClient(t, "/api", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			})
			_, err := c.ListUsers(t.Context(), ListUsersParams{})
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("err = %v, want *APIError", err)
			}
			if apiErr.Status != tt.status {
				t.Errorf("Status = %d, want %d", apiErr.Status, tt.status)
			}
			if apiErr.Message != tt.wantMessage {
				t.Errorf("Message = %q, want %q", apiErr.Message, tt.wantMessage)
			}
			if !strings.HasPrefix(err.Error(), "list users: ") {
				t.Errorf("err = %q, want list users prefix", err)
			}
			if strings.Contains(err.Error(), sentinelToken) {
				t.Errorf("err leaks token: %q", err)
			}
		})
	}
}
