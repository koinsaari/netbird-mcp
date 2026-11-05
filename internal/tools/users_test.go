package tools

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/koinsaari/netbird-mcp/internal/netbird"
)

type fakeUserAPI struct {
	users  []netbird.User
	err    error
	calls  int
	params netbird.ListUsersParams
}

func (f *fakeUserAPI) ListUsers(_ context.Context, p netbird.ListUsersParams) ([]netbird.User, error) {
	f.calls++
	f.params = p
	return f.users, f.err
}

func TestToListUsersOutput(t *testing.T) {
	tests := []struct {
		name string
		in   []netbird.User
		want string
	}{
		{name: "nil", in: nil, want: `{"users":[]}`},
		{name: "empty", in: []netbird.User{}, want: `{"users":[]}`},
		{
			name: "nil auto_groups and optional fields absent",
			in:   []netbird.User{{ID: "u1", Email: "a@x.io", Name: "A", Role: "admin", Status: "active"}},
			want: `{"users":[{"id":"u1","email":"a@x.io","name":"A","role":"admin","status":"active","auto_groups":[],"is_service_user":false,"is_blocked":false,"pending_approval":false}]}`,
		},
		{
			name: "all fields, dropped fields gone, order kept",
			in: []netbird.User{
				{
					ID: "u2", Email: "b@x.io", Name: "B", Role: "user", Status: "invited", LastLogin: "2026-01-02T03:04:05Z",
					AutoGroups: []string{"g2", "g1"}, IsCurrent: true, IsServiceUser: true, IsBlocked: true, PendingApproval: true,
					Issued: "api", IdpID: "idp-secret",
				},
				{ID: "u1", AutoGroups: []string{}},
			},
			want: `{"users":[{"id":"u2","email":"b@x.io","name":"B","role":"user","status":"invited","last_login":"2026-01-02T03:04:05Z","auto_groups":["g2","g1"],"is_service_user":true,"is_blocked":true,"pending_approval":true,"is_current":true},` +
				`{"id":"u1","email":"","name":"","role":"","status":"","auto_groups":[],"is_service_user":false,"is_blocked":false,"pending_approval":false}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(toListUsersOutput(tt.in))
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
			for _, dropped := range []string{"permissions", "idp_id", "issued", "idp-secret"} {
				if strings.Contains(string(got), dropped) {
					t.Errorf("dropped field %q in output: %s", dropped, got)
				}
			}
		})
	}
}

func TestListUsersTool(t *testing.T) {
	yes, no := true, false
	tests := []struct {
		name       string
		args       map[string]any
		api        *fakeUserAPI
		wantErr    string
		wantFilter *bool
		wantUsers  int
	}{
		{
			name:      "success",
			api:       &fakeUserAPI{users: []netbird.User{{ID: "a", IsCurrent: true, LastLogin: "t"}, {ID: "b"}}},
			wantUsers: 2,
		},
		{name: "empty", api: &fakeUserAPI{}},
		{name: "filter true", args: map[string]any{"service_user": true}, api: &fakeUserAPI{}, wantFilter: &yes},
		{name: "filter false", args: map[string]any{"service_user": false}, api: &fakeUserAPI{}, wantFilter: &no},
		{
			name:    "401",
			api:     &fakeUserAPI{err: &netbird.APIError{Status: 401, Message: "bad token"}},
			wantErr: "netbird api: 401: bad token (hint: token invalid or expired)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cs, logs := connectFake(t, fakeClient{userAPI: tt.api})
			res, text := callTool(t, cs, "list_users", tt.args)

			if tt.api.calls != 1 {
				t.Errorf("api calls = %d, want 1", tt.api.calls)
			}
			if !reflect.DeepEqual(tt.api.params.ServiceUser, tt.wantFilter) {
				t.Errorf("service_user = %v, want %v", tt.api.params.ServiceUser, tt.wantFilter)
			}
			if tt.wantErr != "" {
				if !res.IsError || text != tt.wantErr {
					t.Errorf("IsError=%v text=%q, want error %q", res.IsError, text, tt.wantErr)
				}
				if !strings.Contains(logs.String(), "level=WARN") || !strings.Contains(logs.String(), "is_error=true") {
					t.Errorf("want Warn log with is_error=true, got %s", logs)
				}
			} else {
				if res.IsError {
					t.Fatalf("unexpected tool error: %s", text)
				}
				sc, ok := res.StructuredContent.(map[string]any)
				if !ok {
					t.Fatalf("structured content = %#v", res.StructuredContent)
				}
				users, ok := sc["users"].([]any)
				if !ok || len(users) != tt.wantUsers {
					t.Errorf("users = %#v, want %d non-nil", sc["users"], tt.wantUsers)
				}
				if tt.wantUsers == 2 {
					first := users[0].(map[string]any)
					second := users[1].(map[string]any)
					if first["is_current"] != true || first["last_login"] != "t" {
						t.Errorf("first user = %#v", first)
					}
					for _, k := range []string{"is_current", "last_login"} {
						if _, has := second[k]; has {
							t.Errorf("%q present when empty: %#v", k, second)
						}
					}
					if g, ok := second["auto_groups"].([]any); !ok || len(g) != 0 {
						t.Errorf("auto_groups = %#v, want empty non-nil", second["auto_groups"])
					}
				}
				if !strings.Contains(logs.String(), "tool=list_users") || !strings.Contains(logs.String(), "is_error=false") {
					t.Errorf("logs missing tool fields: %s", logs)
				}
			}
			if strings.Contains(text, sentinelToken) || strings.Contains(logs.String(), sentinelToken) {
				t.Errorf("sentinel leaked: text=%q logs=%s", text, logs)
			}
		})
	}
}

func TestListUsersToolSchema(t *testing.T) {
	t.Parallel()
	tool, _ := listUsers(&fakeUserAPI{})
	if tool.Name != "list_users" {
		t.Errorf("name = %q", tool.Name)
	}
	const wantDesc = "Lists users in the NetBird account with role, status and auto-assigned groups. Set service_user to true to list only service users, or false for only regular users."
	if tool.Description != wantDesc {
		t.Errorf("description = %q", tool.Description)
	}
	f, _ := reflect.TypeFor[listUsersInput]().FieldByName("ServiceUser")
	if f.Tag.Get("json") != "service_user,omitempty" || f.Tag.Get("jsonschema") == "" {
		t.Errorf("service_user tag = %q", f.Tag)
	}
	a := tool.Annotations
	if a == nil || !a.ReadOnlyHint || !a.IdempotentHint || a.DestructiveHint != nil || a.OpenWorldHint != nil {
		t.Errorf("annotations = %+v", a)
	}
}
