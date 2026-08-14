package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/oat431/7-solution-interview/internal/application"
	"github.com/oat431/7-solution-interview/internal/infrastructure/auth"
	"github.com/oat431/7-solution-interview/internal/infrastructure/httpapi"
	"github.com/oat431/7-solution-interview/testutil"
)

const testSecret = "0123456789abcdef0123456789abcdef"

type testEnv struct {
	router http.Handler
	repo   *testutil.FakeUserRepository
	tokens *auth.JWTManager
	logs   *bytes.Buffer
}

func newTestEnv() *testEnv {
	repo := testutil.NewFakeUserRepository()
	hasher := testutil.FakeHasher{}
	users := application.NewUserService(repo, hasher)
	tokens := auth.NewJWTManager([]byte(testSecret), time.Hour)
	authSvc := application.NewAuthService(repo, hasher, tokens)
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	return &testEnv{
		router: httpapi.NewRouter(log, users, authSvc),
		repo:   repo,
		tokens: tokens,
		logs:   &buf,
	}
}

func (e *testEnv) do(method, path, body, token string) *httptest.ResponseRecorder {
	var rd *strings.Reader
	if body == "" {
		rd = strings.NewReader("")
	} else {
		rd = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rd)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

func (e *testEnv) register(t *testing.T, name, email, password string) map[string]any {
	t.Helper()
	body := `{"name":"` + name + `","email":"` + email + `","password":"` + password + `"}`
	rec := e.do(http.MethodPost, "/api/v1/auth/register", body, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	return decodeMap(t, rec.Body)
}

func (e *testEnv) login(t *testing.T, email, password string) string {
	t.Helper()
	body := `{"email":"` + email + `","password":"` + password + `"}`
	rec := e.do(http.MethodPost, "/api/v1/auth/login", body, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	return decodeMap(t, rec.Body)["token"].(string)
}

func decodeMap(t *testing.T, b *bytes.Buffer) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(b.Bytes(), &m); err != nil {
		t.Fatalf("decode response: %v (body: %s)", err, b.String())
	}
	return m
}

func decodeErr(t *testing.T, b *bytes.Buffer) map[string]any {
	t.Helper()
	body := decodeMap(t, b)
	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error envelope, got %s", b.String())
	}
	return errObj
}

// ---- Register (FR-001) ----

func TestRegisterReturns201WithoutPasswordMaterial(t *testing.T) {
	e := newTestEnv()
	rec := e.do(http.MethodPost, "/api/v1/auth/register",
		`{"name":"Ada Lovelace","email":"ada@example.com","password":"s3cret-pass"}`, "")

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	raw := rec.Body.String()
	if strings.Contains(raw, "password") || strings.Contains(raw, "s3cret-pass") {
		t.Fatalf("response leaks password material: %s", raw)
	}
	m := decodeMap(t, rec.Body)
	for _, key := range []string{"id", "name", "email", "createdAt"} {
		if _, ok := m[key]; !ok {
			t.Fatalf("missing field %q in %s", key, raw)
		}
	}
}

func TestRegisterValidationError(t *testing.T) {
	e := newTestEnv()
	rec := e.do(http.MethodPost, "/api/v1/auth/register",
		`{"name":"Ada","email":"not-an-email","password":"s3cret-pass"}`, "")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	errObj := decodeErr(t, rec.Body)
	if errObj["code"] != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %v", errObj["code"])
	}
	details, ok := errObj["details"].([]any)
	if !ok || len(details) == 0 {
		t.Fatalf("expected field details, got %s", rec.Body.String())
	}
}

func TestRegisterDuplicateEmail409(t *testing.T) {
	e := newTestEnv()
	e.register(t, "Ada", "ada@example.com", "s3cret-pass")

	rec := e.do(http.MethodPost, "/api/v1/auth/register",
		`{"name":"Ada Two","email":"ada@example.com","password":"s3cret-pass"}`, "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
	if decodeErr(t, rec.Body)["code"] != "EMAIL_ALREADY_EXISTS" {
		t.Fatalf("expected EMAIL_ALREADY_EXISTS, got %s", rec.Body.String())
	}
}

func TestRegisterMalformedJSON400(t *testing.T) {
	e := newTestEnv()
	rec := e.do(http.MethodPost, "/api/v1/auth/register", `{"name":`, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestRegisterUnknownFieldsRejected(t *testing.T) {
	e := newTestEnv()
	rec := e.do(http.MethodPost, "/api/v1/auth/register",
		`{"name":"Ada","email":"ada@example.com","password":"s3cret-pass","admin":true}`, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown field, got %d", rec.Code)
	}
}

// ---- Login (FR-002) ----

func TestLoginHappyPathAndTokenUsage(t *testing.T) {
	e := newTestEnv()
	e.register(t, "Ada", "ada@example.com", "s3cret-pass")

	token := e.login(t, "ada@example.com", "s3cret-pass")

	rec := e.do(http.MethodGet, "/api/v1/users", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("protected list with valid token: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestLoginResponseShape(t *testing.T) {
	e := newTestEnv()
	e.register(t, "Ada", "ada@example.com", "s3cret-pass")

	rec := e.do(http.MethodPost, "/api/v1/auth/login",
		`{"email":"ada@example.com","password":"s3cret-pass"}`, "")
	m := decodeMap(t, rec.Body)
	if m["tokenType"] != "Bearer" {
		t.Fatalf("expected tokenType Bearer, got %v", m["tokenType"])
	}
	if m["expiresIn"].(float64) != 3600 {
		t.Fatalf("expected expiresIn 3600, got %v", m["expiresIn"])
	}
}

func TestLoginWrongPasswordAndUnknownEmailIdentical401(t *testing.T) {
	e := newTestEnv()
	e.register(t, "Ada", "ada@example.com", "s3cret-pass")

	wrongPw := e.do(http.MethodPost, "/api/v1/auth/login",
		`{"email":"ada@example.com","password":"nope-nope"}`, "")
	unknownEmail := e.do(http.MethodPost, "/api/v1/auth/login",
		`{"email":"ghost@example.com","password":"whatever"}`, "")

	if wrongPw.Code != http.StatusUnauthorized || unknownEmail.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401/401, got %d/%d", wrongPw.Code, unknownEmail.Code)
	}
	if decodeErr(t, wrongPw.Body)["code"] != "INVALID_CREDENTIALS" {
		t.Fatalf("expected INVALID_CREDENTIALS, got %s", wrongPw.Body.String())
	}
	if wrongPw.Body.String() != unknownEmail.Body.String() {
		t.Fatal("wrong-password and unknown-email responses must be identical (no user enumeration)")
	}
}

// ---- Auth middleware (FR-002/003/004) ----

func TestProtectedEndpointsRejectMissingToken(t *testing.T) {
	e := newTestEnv()
	cases := []struct{ method, path string }{
		{http.MethodPost, "/api/v1/users"},
		{http.MethodGet, "/api/v1/users"},
		{http.MethodGet, "/api/v1/users/665f1c2d3e4f5a6b7c8d9e0f"},
		{http.MethodPut, "/api/v1/users/665f1c2d3e4f5a6b7c8d9e0f"},
		{http.MethodDelete, "/api/v1/users/665f1c2d3e4f5a6b7c8d9e0f"},
	}
	for _, c := range cases {
		rec := e.do(c.method, c.path, "", "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s without token: expected 401, got %d", c.method, c.path, rec.Code)
		}
	}
}

func TestProtectedEndpointsRejectGarbageToken(t *testing.T) {
	e := newTestEnv()
	rec := e.do(http.MethodGet, "/api/v1/users", "", "not.a.jwt")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestProtectedEndpointsRejectExpiredToken(t *testing.T) {
	e := newTestEnv()
	expiredMgr := auth.NewJWTManager([]byte(testSecret), -time.Minute)
	expired, _, err := expiredMgr.Issue(context.Background(), application.TokenClaims{Subject: "sub", Email: "ada@example.com"})
	if err != nil {
		t.Fatalf("issue expired token: %v", err)
	}
	rec := e.do(http.MethodGet, "/api/v1/users", "", expired)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired token, got %d", rec.Code)
	}
}

// ---- CRUD (FR-003..FR-007) ----

func TestCreateUserProtected(t *testing.T) {
	e := newTestEnv()
	first := e.register(t, "Ada", "ada@example.com", "s3cret-pass")
	token := e.login(t, "ada@example.com", "s3cret-pass")

	rec := e.do(http.MethodPost, "/api/v1/users",
		`{"name":"Grace Hopper","email":"grace@example.com","password":"c0bol-rul3z"}`, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	m := decodeMap(t, rec.Body)
	if m["id"] == first["id"] {
		t.Fatal("expected a new user id")
	}
}

func TestGetUser(t *testing.T) {
	e := newTestEnv()
	created := e.register(t, "Ada", "ada@example.com", "s3cret-pass")
	token := e.login(t, "ada@example.com", "s3cret-pass")

	rec := e.do(http.MethodGet, "/api/v1/users/"+created["id"].(string), "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if decodeMap(t, rec.Body)["email"] != "ada@example.com" {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestGetUserNotFound(t *testing.T) {
	e := newTestEnv()
	e.register(t, "Ada", "ada@example.com", "s3cret-pass")
	token := e.login(t, "ada@example.com", "s3cret-pass")

	rec := e.do(http.MethodGet, "/api/v1/users/665f1c2d3e4f5a6b7c8d9e0f", "", token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if decodeErr(t, rec.Body)["code"] != "USER_NOT_FOUND" {
		t.Fatalf("expected USER_NOT_FOUND, got %s", rec.Body.String())
	}
}

func TestGetUserInvalidID(t *testing.T) {
	e := newTestEnv()
	e.register(t, "Ada", "ada@example.com", "s3cret-pass")
	token := e.login(t, "ada@example.com", "s3cret-pass")

	rec := e.do(http.MethodGet, "/api/v1/users/zzz", "", token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if decodeErr(t, rec.Body)["code"] != "INVALID_ID" {
		t.Fatalf("expected INVALID_ID, got %s", rec.Body.String())
	}
}

func TestListUsers(t *testing.T) {
	e := newTestEnv()
	e.register(t, "Ada", "ada@example.com", "s3cret-pass")
	token := e.login(t, "ada@example.com", "s3cret-pass")
	e.do(http.MethodPost, "/api/v1/users",
		`{"name":"Grace","email":"grace@example.com","password":"c0bol-rul3z"}`, token)

	rec := e.do(http.MethodGet, "/api/v1/users", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	m := decodeMap(t, rec.Body)
	data := m["data"].([]any)
	if len(data) != 2 {
		t.Fatalf("expected 2 users, got %d (%s)", len(data), rec.Body.String())
	}
	meta := m["meta"].(map[string]any)
	if meta["count"].(float64) != 2 {
		t.Fatalf("expected count 2, got %v", meta["count"])
	}
}

func TestUpdateUserNameOnly(t *testing.T) {
	e := newTestEnv()
	created := e.register(t, "Ada", "ada@example.com", "s3cret-pass")
	token := e.login(t, "ada@example.com", "s3cret-pass")

	rec := e.do(http.MethodPut, "/api/v1/users/"+created["id"].(string),
		`{"name":"Ada Byron"}`, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	m := decodeMap(t, rec.Body)
	if m["name"] != "Ada Byron" || m["email"] != "ada@example.com" {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestUpdateEmptyBodyRejected(t *testing.T) {
	e := newTestEnv()
	created := e.register(t, "Ada", "ada@example.com", "s3cret-pass")
	token := e.login(t, "ada@example.com", "s3cret-pass")

	rec := e.do(http.MethodPut, "/api/v1/users/"+created["id"].(string), `{}`, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdatePasswordFieldRejected(t *testing.T) {
	e := newTestEnv()
	created := e.register(t, "Ada", "ada@example.com", "s3cret-pass")
	token := e.login(t, "ada@example.com", "s3cret-pass")

	rec := e.do(http.MethodPut, "/api/v1/users/"+created["id"].(string),
		`{"password":"hacked"}`, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for password field, got %d", rec.Code)
	}
}

func TestUpdateEmailConflict409(t *testing.T) {
	e := newTestEnv()
	created := e.register(t, "Ada", "ada@example.com", "s3cret-pass")
	token := e.login(t, "ada@example.com", "s3cret-pass")
	e.do(http.MethodPost, "/api/v1/users",
		`{"name":"Grace","email":"grace@example.com","password":"c0bol-rul3z"}`, token)

	rec := e.do(http.MethodPut, "/api/v1/users/"+created["id"].(string),
		`{"email":"grace@example.com"}`, token)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestUpdateNotFound(t *testing.T) {
	e := newTestEnv()
	e.register(t, "Ada", "ada@example.com", "s3cret-pass")
	token := e.login(t, "ada@example.com", "s3cret-pass")

	rec := e.do(http.MethodPut, "/api/v1/users/665f1c2d3e4f5a6b7c8d9e0f", `{"name":"X"}`, token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestDeleteThenGet404(t *testing.T) {
	e := newTestEnv()
	created := e.register(t, "Ada", "ada@example.com", "s3cret-pass")
	token := e.login(t, "ada@example.com", "s3cret-pass")

	rec := e.do(http.MethodDelete, "/api/v1/users/"+created["id"].(string), "", token)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	rec = e.do(http.MethodGet, "/api/v1/users/"+created["id"].(string), "", token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", rec.Code)
	}
}

func TestDeleteNotFound(t *testing.T) {
	e := newTestEnv()
	e.register(t, "Ada", "ada@example.com", "s3cret-pass")
	token := e.login(t, "ada@example.com", "s3cret-pass")

	rec := e.do(http.MethodDelete, "/api/v1/users/665f1c2d3e4f5a6b7c8d9e0f", "", token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// ---- Observability (FR-008) ----

func TestHealthz(t *testing.T) {
	e := newTestEnv()
	rec := e.do(http.MethodGet, "/healthz", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestLoggingMiddlewareEmitsStructuredLine(t *testing.T) {
	e := newTestEnv()
	e.do(http.MethodGet, "/healthz", "", "")

	out := e.logs.String()
	if !strings.Contains(out, `"method":"GET"`) || !strings.Contains(out, `"path":"/healthz"`) {
		t.Fatalf("expected structured log with method/path, got: %s", out)
	}
	if !strings.Contains(out, `"status":200`) || !strings.Contains(out, `"duration_ms"`) {
		t.Fatalf("expected status/duration in log, got: %s", out)
	}
}

func TestLoggingMiddlewareNeverLogsSecrets(t *testing.T) {
	e := newTestEnv()
	e.do(http.MethodGet, "/api/v1/users", "", "super.secret.token")
	if strings.Contains(e.logs.String(), "super.secret.token") {
		t.Fatalf("log must not contain the bearer token: %s", e.logs.String())
	}
}

// ---- Method semantics ----

func TestWrongMethodOnKnownPath405(t *testing.T) {
	e := newTestEnv()
	rec := e.do(http.MethodDelete, "/api/v1/auth/register", "", "")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
