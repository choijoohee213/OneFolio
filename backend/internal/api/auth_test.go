package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const testToken = "s3cret"

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func statusFor(t *testing.T, gate http.Handler, path, token string) int {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		request.Header.Set(TokenHeader, token)
	}
	recorder := httptest.NewRecorder()
	gate.ServeHTTP(recorder, request)
	return recorder.Code
}

func TestAuthRequiresToken(t *testing.T) {
	gate := Auth(testToken, okHandler())

	cases := []struct {
		name  string
		path  string
		token string
		want  int
	}{
		{"헬스체크는 암호 없이 열린다", "/health", "", http.StatusOK},
		{"암호가 없으면 막는다", "/api/portfolio", "", http.StatusUnauthorized},
		{"암호가 틀리면 막는다", "/api/portfolio", "wrong", http.StatusUnauthorized},
		{"길이만 같아도 막는다", "/api/portfolio", "s3cre7", http.StatusUnauthorized},
		{"암호가 맞으면 통과한다", "/api/portfolio", testToken, http.StatusOK},
		{"OCR 도 막는다", "/api/ocr", "", http.StatusUnauthorized},
		{"시세도 막는다", "/api/quotes", "", http.StatusUnauthorized},
		{"확인용 엔드포인트도 막는다", "/api/auth", "", http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := statusFor(t, gate, tc.path, tc.token); got != tc.want {
				t.Errorf("%s: %d, want %d", tc.path, got, tc.want)
			}
		})
	}
}

// APP_TOKEN 이 비면 게이트를 걸지 않는다. 로컬 개발에서 암호 없이 돌리기 위한 것이다.
func TestAuthDisabledWithoutToken(t *testing.T) {
	gate := Auth("", okHandler())
	if got := statusFor(t, gate, "/api/portfolio", ""); got != http.StatusOK {
		t.Errorf("암호 없이 통과해야 한다: %d", got)
	}
}
