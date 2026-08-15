package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/oat431/7-solution-interview/internal/application"
	"github.com/oat431/7-solution-interview/internal/infrastructure/auth"
	"github.com/oat431/7-solution-interview/internal/infrastructure/httpapi"
	"github.com/oat431/7-solution-interview/testutil"
)

const testSecret = "0123456789abcdef0123456789abcdef"

type testEnv struct {
	app    *fiber.App
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
		app:    httpapi.NewApp(log, users, authSvc),
		repo:   repo,
		tokens: tokens,
		logs:   &buf,
	}
}

// do performs a request through Fiber's in-process test facility.
func (e *testEnv) do(t *testing.T, method, path, body, token string) (int, string) {
	t.Helper()
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rd)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("%s %s: app.Test: %v", method, path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("%s %s: read body: %v", method, path, err)
	}
	return resp.StatusCode, string(raw)
}

func (e *testEnv) register(t *testing.T, name, email, password string) map[string]any {
	t.Helper()
	body := `{"name":"` + name + `","email":"` + email + `","password":"` + password + `"}`
	code, raw := e.do(t, http.MethodPost, "/api/v1/auth/register", body, "")
	if code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d (%s)", code, raw)
	}
	return decodeMap(t, raw)
}

func (e *testEnv) login(t *testing.T, email, password string) string {
	t.Helper()
	body := `{"email":"` + email + `","password":"` + password + `"}`
	code, raw := e.do(t, http.MethodPost, "/api/v1/auth/login", body, "")
	if code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d (%s)", code, raw)
	}
	return decodeMap(t, raw)["token"].(string)
}

func decodeMap(t *testing.T, raw string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("decode response: %v (body: %s)", err, raw)
	}
	return m
}

func decodeErr(t *testing.T, raw string) map[string]any {
	t.Helper()
	body := decodeMap(t, raw)
	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error envelope, got %s", raw)
	}
	return errObj
}

// ---- Register (FR-001) ----

func TestRegisterReturns201WithoutPasswordMaterial(t *testing.T) {
	e := newTestEnv()
	code, raw := e.do(t, http.MethodPost, "/api/v1/auth/register",
		`{"name":"Ada Lovelace","email":"ada@example.com","password":"s3cret-pass"}`, "")

	if code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", code)
	}
	if strings.Contains(raw, "password") || strings.Contains(raw, "s3cret-pass") {
		t.Fatalf("response leaks password material: %s", raw)
	}
	m := decodeMap(t, raw)
	for _, key := range []string{"id", "name", "email", "createdAt"} {
		if _, ok := m[key]; !ok {
			t.Fatalf("missing field %q in %s", key, raw)
		}
	}
}

func TestRegisterValidationError(t *testing.T) {
	e := newTestEnv()
	code, raw := e.do(t, http.MethodPost, "/api/v1/auth/register",
		`{"name":"Ada","email":"not-an-email","password":"s3cret-pass"}`, "")

	if code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", code)
	}
	errObj := decodeErr(t, raw)
	if errObj["code"] != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %v", errObj["code"])
	}
	details, ok := errObj["details"].([]any)
	if !ok || len(details) == 0 {
		t.Fatalf("expected field details, got %s", raw)
	}
}

func TestRegisterDuplicateEmail409(t *testing.T) {
	e := newTestEnv()
	e.register(t, "Ada", "ada@example.com", "s3cret-pass")

	code, raw := e.do(t, http.MethodPost, "/api/v1/auth/register",
		`{"name":"Ada Two","email":"ada@example.com","password":"s3cret-pass"}`, "")
	if code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", code)
	}
	if decodeErr(t, raw)["code"] != "EMAIL_ALREADY_EXISTS" {
		t.Fatalf("expected EMAIL_ALREADY_EXISTS, got %s", raw)
	}
}

func TestRegisterMalformedJSON400(t *testing.T) {
	e := newTestEnv()
	code, _ := e.do(t, http.MethodPost, "/api/v1/auth/register", `{"name":`, "")
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", code)
	}
}

func TestRegisterUnknownFieldsRejected(t *testing.T) {
	e := newTestEnv()
	code, _ := e.do(t, http.MethodPost, "/api/v1/auth/register",
		`{"name":"Ada","email":"ada@example.com","password":"s3cret-pass","admin":true}`, "")
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown field, got %d", code)
	}
}

// ---- Login (FR-002) ----

func TestLoginHappyPathAndTokenUsage(t *testing.T) {
	e := newTestEnv()
	e.register(t, "Ada", "ada@example.com", "s3cret-pass")

	token := e.login(t, "ada@example.com", "s3cret-pass")

	code, raw := e.do(t, http.MethodGet, "/api/v1/users", "", token)
	if code != http.StatusOK {
		t.Fatalf("protected list with valid token: expected 200, got %d (%s)", code, raw)
	}
}

func TestLoginResponseShape(t *testing.T) {
	e := newTestEnv()
	e.register(t, "Ada", "ada@example.com", "s3cret-pass")

	code, raw := e.do(t, http.MethodPost, "/api/v1/auth/login",
		`{"email":"ada@example.com","password":"s3cret-pass"}`, "")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	m := decodeMap(t, raw)
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

	_, wrongPw := e.do(t, http.MethodPost, "/api/v1/auth/login",
		`{"email":"ada@example.com","password":"nope-nope"}`, "")
	_, unknownEmail := e.do(t, http.MethodPost, "/api/v1/auth/login",
		`{"email":"ghost@example.com","password":"whatever"}`, "")

	if !strings.Contains(wrongPw, `"code":"INVALID_CREDENTIALS"`) {
		t.Fatalf("expected INVALID_CREDENTIALS, got %s", wrongPw)
	}
	if wrongPw != unknownEmail {
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
		code, _ := e.do(t, c.method, c.path, "", "")
		if code != http.StatusUnauthorized {
			t.Fatalf("%s %s without token: expected 401, got %d", c.method, c.path, code)
		}
	}
}

func TestProtectedEndpointsRejectGarbageToken(t *testing.T) {
	e := newTestEnv()
	code, _ := e.do(t, http.MethodGet, "/api/v1/users", "", "not.a.jwt")
	if code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", code)
	}
}

func TestProtectedEndpointsRejectExpiredToken(t *testing.T) {
	e := newTestEnv()
	expiredMgr := auth.NewJWTManager([]byte(testSecret), -time.Minute)
	expired, _, err := expiredMgr.Issue(context.Background(), application.TokenClaims{Subject: "sub", Email: "ada@example.com"})
	if err != nil {
		t.Fatalf("issue expired token: %v", err)
	}
	code, _ := e.do(t, http.MethodGet, "/api/v1/users", "", expired)
	if code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired token, got %d", code)
	}
}

// ---- CRUD (FR-003..FR-007) ----

func TestCreateUserProtected(t *testing.T) {
	e := newTestEnv()
	first := e.register(t, "Ada", "ada@example.com", "s3cret-pass")
	token := e.login(t, "ada@example.com", "s3cret-pass")

	code, raw := e.do(t, http.MethodPost, "/api/v1/users",
		`{"name":"Grace Hopper","email":"grace@example.com","password":"c0bol-rul3z"}`, token)
	if code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (%s)", code, raw)
	}
	m := decodeMap(t, raw)
	if m["id"] == first["id"] {
		t.Fatal("expected a new user id")
	}
}

func TestGetUser(t *testing.T) {
	e := newTestEnv()
	created := e.register(t, "Ada", "ada@example.com", "s3cret-pass")
	token := e.login(t, "ada@example.com", "s3cret-pass")

	code, raw := e.do(t, http.MethodGet, "/api/v1/users/"+created["id"].(string), "", token)
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if decodeMap(t, raw)["email"] != "ada@example.com" {
		t.Fatalf("unexpected body: %s", raw)
	}
}

func TestGetUserNotFound(t *testing.T) {
	e := newTestEnv()
	e.register(t, "Ada", "ada@example.com", "s3cret-pass")
	token := e.login(t, "ada@example.com", "s3cret-pass")

	code, raw := e.do(t, http.MethodGet, "/api/v1/users/665f1c2d3e4f5a6b7c8d9e0f", "", token)
	if code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", code)
	}
	if decodeErr(t, raw)["code"] != "USER_NOT_FOUND" {
		t.Fatalf("expected USER_NOT_FOUND, got %s", raw)
	}
}

func TestGetUserInvalidID(t *testing.T) {
	e := newTestEnv()
	e.register(t, "Ada", "ada@example.com", "s3cret-pass")
	token := e.login(t, "ada@example.com", "s3cret-pass")

	code, raw := e.do(t, http.MethodGet, "/api/v1/users/zzz", "", token)
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", code)
	}
	if decodeErr(t, raw)["code"] != "INVALID_ID" {
		t.Fatalf("expected INVALID_ID, got %s", raw)
	}
}

func TestListUsers(t *testing.T) {
	e := newTestEnv()
	e.register(t, "Ada", "ada@example.com", "s3cret-pass")
	token := e.login(t, "ada@example.com", "s3cret-pass")
	e.do(t, http.MethodPost, "/api/v1/users",
		`{"name":"Grace","email":"grace@example.com","password":"c0bol-rul3z"}`, token)

	code, raw := e.do(t, http.MethodGet, "/api/v1/users", "", token)
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	m := decodeMap(t, raw)
	data := m["data"].([]any)
	if len(data) != 2 {
		t.Fatalf("expected 2 users, got %d (%s)", len(data), raw)
	}
	meta := m["meta"].(map[string]any)
	if meta["count"].(float64) != 2 {
		t.Fatalf("expected count 2, got %v", meta["count"])
	}
}

// AC-005b: an empty database lists as {"data":[], "meta":{"count":0}} —
// never {"data":null}. The token is issued directly so no user is created.
func TestListUsersEmpty(t *testing.T) {
	e := newTestEnv()
	tk, _, err := e.tokens.Issue(context.Background(), application.TokenClaims{Subject: "seed", Email: "seed@example.com"})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	code, raw := e.do(t, http.MethodGet, "/api/v1/users", "", tk)
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", code, raw)
	}
	if strings.Contains(raw, `"data":null`) {
		t.Fatalf("empty list must serialize as [], not null: %s", raw)
	}
	m := decodeMap(t, raw)
	data := m["data"].([]any)
	if len(data) != 0 {
		t.Fatalf("expected empty data, got %d", len(data))
	}
	meta := m["meta"].(map[string]any)
	if meta["count"].(float64) != 0 {
		t.Fatalf("expected count 0, got %v", meta["count"])
	}
}

func TestUpdateUserNameOnly(t *testing.T) {
	e := newTestEnv()
	created := e.register(t, "Ada", "ada@example.com", "s3cret-pass")
	tok := e.login(t, "ada@example.com", "s3cret-pass")

	code, raw := e.do(t, http.MethodPut, "/api/v1/users/"+created["id"].(string),
		`{"name":"Ada Byron"}`, tok)
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", code, raw)
	}
	m := decodeMap(t, raw)
	if m["name"] != "Ada Byron" || m["email"] != "ada@example.com" {
		t.Fatalf("unexpected body: %s", raw)
	}
}

// AC-006b: email-only partial update succeeds; name stays untouched.
func TestUpdateUserEmailOnly(t *testing.T) {
	e := newTestEnv()
	created := e.register(t, "Ada", "ada@example.com", "s3cret-pass")
	tk := e.login(t, "ada@example.com", "s3cret-pass")

	code, raw := e.do(t, http.MethodPut, "/api/v1/users/"+created["id"].(string),
		`{"email":"ada.byron@example.com"}`, tk)
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", code, raw)
	}
	m := decodeMap(t, raw)
	if m["email"] != "ada.byron@example.com" || m["name"] != "Ada" {
		t.Fatalf("unexpected body: %s", raw)
	}
}

func TestUpdateEmptyBodyRejected(t *testing.T) {
	e := newTestEnv()
	created := e.register(t, "Ada", "ada@example.com", "s3cret-pass")
	token := e.login(t, "ada@example.com", "s3cret-pass")

	code, _ := e.do(t, http.MethodPut, "/api/v1/users/"+created["id"].(string), `{}`, token)
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", code)
	}
}

func TestUpdatePasswordFieldRejected(t *testing.T) {
	e := newTestEnv()
	created := e.register(t, "Ada", "ada@example.com", "s3cret-pass")
	token := e.login(t, "ada@example.com", "s3cret-pass")

	code, _ := e.do(t, http.MethodPut, "/api/v1/users/"+created["id"].(string),
		`{"password":"hacked"}`, token)
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400 for password field, got %d", code)
	}
}

func TestUpdateEmailConflict409(t *testing.T) {
	e := newTestEnv()
	created := e.register(t, "Ada", "ada@example.com", "s3cret-pass")
	token := e.login(t, "ada@example.com", "s3cret-pass")
	e.do(t, http.MethodPost, "/api/v1/users",
		`{"name":"Grace","email":"grace@example.com","password":"c0bol-rul3z"}`, token)

	code, raw := e.do(t, http.MethodPut, "/api/v1/users/"+created["id"].(string),
		`{"email":"grace@example.com"}`, token)
	if code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (%s)", code, raw)
	}
}

func TestUpdateNotFound(t *testing.T) {
	e := newTestEnv()
	e.register(t, "Ada", "ada@example.com", "s3cret-pass")
	token := e.login(t, "ada@example.com", "s3cret-pass")

	code, _ := e.do(t, http.MethodPut, "/api/v1/users/665f1c2d3e4f5a6b7c8d9e0f", `{"name":"X"}`, token)
	if code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", code)
	}
}

func TestDeleteThenGet404(t *testing.T) {
	e := newTestEnv()
	created := e.register(t, "Ada", "ada@example.com", "s3cret-pass")
	token := e.login(t, "ada@example.com", "s3cret-pass")

	code, _ := e.do(t, http.MethodDelete, "/api/v1/users/"+created["id"].(string), "", token)
	if code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", code)
	}

	code, _ = e.do(t, http.MethodGet, "/api/v1/users/"+created["id"].(string), "", token)
	if code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", code)
	}
}

func TestDeleteNotFound(t *testing.T) {
	e := newTestEnv()
	e.register(t, "Ada", "ada@example.com", "s3cret-pass")
	token := e.login(t, "ada@example.com", "s3cret-pass")

	code, _ := e.do(t, http.MethodDelete, "/api/v1/users/665f1c2d3e4f5a6b7c8d9e0f", "", token)
	if code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", code)
	}
}

// ---- Observability (FR-008) ----

func TestHealthz(t *testing.T) {
	e := newTestEnv()
	code, _ := e.do(t, http.MethodGet, "/healthz", "", "")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
}

func TestLoggingMiddlewareEmitsStructuredLine(t *testing.T) {
	e := newTestEnv()
	e.do(t, http.MethodGet, "/healthz", "", "")

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
	e.do(t, http.MethodGet, "/api/v1/users", "", "super.secret.token")
	if strings.Contains(e.logs.String(), "super.secret.token") {
		t.Fatalf("log must not contain the bearer token: %s", e.logs.String())
	}
}

func TestUnknownRoute404(t *testing.T) {
	e := newTestEnv()
	code, raw := e.do(t, http.MethodGet, "/api/v1/nope", "", "")
	if code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", code)
	}
	if decodeErr(t, raw)["code"] != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND envelope, got %s", raw)
	}
}

// ---- Method semantics ----

func TestWrongMethodOnKnownPath405(t *testing.T) {
	e := newTestEnv()
	code, _ := e.do(t, http.MethodDelete, "/api/v1/auth/register", "", "")
	if code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", code)
	}
}
