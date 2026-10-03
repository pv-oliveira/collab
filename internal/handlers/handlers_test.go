package handlers

import (
	"apis/internal/middleware"
	"apis/internal/repositories"
	"apis/internal/services"
	"apis/internal/testutil"
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const secret = "test-secret"

// newRouter monta as mesmas rotas do main.go com dependências reais e o
// Postgres de teste, para exercitar a conversão erro → status HTTP.
func newRouter(t *testing.T, db *sql.DB) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	tokens := testutil.Tokens(t, secret)
	auth := &AuthHandler{Service: &services.AuthService{Repo: &repositories.UserRepository{DB: db}, JWTSecret: secret}, Tokens: tokens}
	docs := &DocumentHandler{Service: &services.DocumentService{Repo: &repositories.DocumentRepository{DB: db}}}

	r := gin.New()
	r.POST("/auth/register", auth.Register)
	r.POST("/auth/login", auth.Login)
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware(tokens))
	protected.POST("/auth/logout", auth.Logout)
	protected.POST("/documents", docs.Create)
	protected.GET("/documents/:id", docs.Get)
	protected.GET("/documents", docs.List)
	protected.PUT("/documents/:id", docs.Update)
	return r
}

func do(t *testing.T, r *gin.Engine, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAuthHandlers(t *testing.T) {
	db := testutil.DB(t)
	r := newRouter(t, db)
	email := uuid.NewString() + "@test.dev"
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM users WHERE email = $1`, email) })
	creds := map[string]string{"email": email, "password": "s3nha"}

	w := do(t, r, http.MethodPost, "/auth/register", "", creds)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: status %d, body %s", w.Code, w.Body)
	}
	if strings.Contains(strings.ToLower(w.Body.String()), "password") || strings.Contains(w.Body.String(), "$2a$") {
		t.Errorf("register expôs a senha: %s", w.Body)
	}

	if w := do(t, r, http.MethodPost, "/auth/register", "", creds); w.Code != http.StatusConflict {
		t.Errorf("register duplicado: status %d, esperava 409 (body %s)", w.Code, w.Body)
	} else if strings.Contains(w.Body.String(), "pq:") {
		t.Errorf("409 vazou erro do Postgres: %s", w.Body)
	}

	if w := do(t, r, http.MethodPost, "/auth/register", "", map[string]string{"email": "x@test.dev"}); w.Code != http.StatusBadRequest {
		t.Errorf("register sem senha: status %d, esperava 400", w.Code)
	}

	if w := do(t, r, http.MethodPost, "/auth/login", "", map[string]string{"email": email, "password": "errada"}); w.Code != http.StatusUnauthorized {
		t.Errorf("login com senha errada: status %d, esperava 401", w.Code)
	}

	w = do(t, r, http.MethodPost, "/auth/login", "", creds)
	var resp struct{ Token string }
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil || w.Code != http.StatusOK || resp.Token == "" {
		t.Fatalf("login: status %d, body %s", w.Code, w.Body)
	}

	if w := do(t, r, http.MethodGet, "/documents", resp.Token, nil); w.Code != http.StatusOK {
		t.Errorf("antes do logout: status %d, esperava 200", w.Code)
	}
	if w := do(t, r, http.MethodPost, "/auth/logout", resp.Token, nil); w.Code != http.StatusNoContent {
		t.Errorf("logout: status %d, esperava 204 (body %s)", w.Code, w.Body)
	}
	if w := do(t, r, http.MethodGet, "/documents", resp.Token, nil); w.Code != http.StatusUnauthorized {
		t.Errorf("depois do logout: status %d, esperava 401", w.Code)
	}
	if w := do(t, r, http.MethodPost, "/auth/logout", resp.Token, nil); w.Code != http.StatusUnauthorized {
		t.Errorf("logout repetido: status %d, esperava 401", w.Code)
	}
}

func TestDocumentHandlersEnforceOwnership(t *testing.T) {
	db := testutil.DB(t)
	r := newRouter(t, db)
	owner := testutil.CreateUser(t, db)
	intruder := testutil.CreateUser(t, db)
	ownerToken, _ := middleware.GenerateToken(secret, owner.ID)
	intruderToken, _ := middleware.GenerateToken(secret, intruder.ID)

	if w := do(t, r, http.MethodGet, "/documents", "", nil); w.Code != http.StatusUnauthorized {
		t.Errorf("sem token: status %d, esperava 401", w.Code)
	}

	w := do(t, r, http.MethodPost, "/documents", ownerToken, map[string]string{"title": "Meu"})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status %d, body %s", w.Code, w.Body)
	}
	var doc struct{ ID string }
	if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	path := "/documents/" + doc.ID

	if w := do(t, r, http.MethodGet, path, ownerToken, nil); w.Code != http.StatusOK {
		t.Errorf("dono lendo: status %d, esperava 200", w.Code)
	}
	if w := do(t, r, http.MethodGet, path, intruderToken, nil); w.Code != http.StatusNotFound {
		t.Errorf("outro usuário lendo: status %d, esperava 404", w.Code)
	}
	if w := do(t, r, http.MethodPut, path, intruderToken, map[string]string{"title": "x", "content": "y"}); w.Code != http.StatusNotFound {
		t.Errorf("outro usuário editando: status %d, esperava 404", w.Code)
	}
	if w := do(t, r, http.MethodGet, "/documents", intruderToken, nil); strings.Contains(w.Body.String(), doc.ID) {
		t.Errorf("listagem de outro usuário contém o documento: %s", w.Body)
	}
}

func TestDocumentHandlersErrors(t *testing.T) {
	db := testutil.DB(t)
	r := newRouter(t, db)

	t.Run("usuário sem documentos recebe []", func(t *testing.T) {
		user := testutil.CreateUser(t, db)
		token, _ := middleware.GenerateToken(secret, user.ID)
		w := do(t, r, http.MethodGet, "/documents", token, nil)
		if w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != "[]" {
			t.Errorf("status %d, body %s; esperava 200 e []", w.Code, w.Body)
		}
	})

	t.Run("criar para usuário inexistente não retorna 201", func(t *testing.T) {
		token, _ := middleware.GenerateToken(secret, uuid.NewString()) // token válido, usuário não existe
		w := do(t, r, http.MethodPost, "/documents", token, map[string]string{"title": "x"})
		if w.Code != http.StatusInternalServerError {
			t.Errorf("status %d, esperava 500 (body %s)", w.Code, w.Body)
		}
		if strings.Contains(w.Body.String(), "pq:") {
			t.Errorf("vazou erro do Postgres: %s", w.Body)
		}
	})

	t.Run("erro no banco ao listar responde internal error", func(t *testing.T) {
		broken, err := sql.Open("postgres", "postgres://u:p@127.0.0.1:1/x?sslmode=disable")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = broken.Close() })
		token, _ := middleware.GenerateToken(secret, uuid.NewString())

		w := do(t, newRouter(t, broken), http.MethodGet, "/documents", token, nil)
		if w.Code != http.StatusInternalServerError || !strings.Contains(w.Body.String(), "internal error") {
			t.Errorf("status %d, body %s; esperava 500 com internal error", w.Code, w.Body)
		}
	})
}
