package api

import (
	"crypto/subtle"
	"net/http"
)

// TokenHeader 는 프론트가 공유 암호를 실어 보내는 헤더다.
const TokenHeader = "X-App-Token"

// openPaths 는 암호 없이 열리는 경로다. /health 는 Render 헬스체크와 콜드스타트
// 예열이 두드리는 곳이라 닫을 수 없다. 자산 데이터도 외부 API 키도 쓰지 않는다.
var openPaths = map[string]bool{"/health": true}

// Auth 는 공유 암호 게이트다. 백엔드가 남의 Gemini·KIS 호출을 대신 쳐 주는
// 공짜 프록시가 되는 것을 막는 게 목적이다. CORS 로는 막지 못한다 — 브라우저만
// 지키는 규칙이라 curl 은 그냥 통과한다.
//
// token 이 비면 게이트를 걸지 않는다(로컬 개발).
//
// CORS 가 이 핸들러를 감싸야 한다. 프리플라이트(OPTIONS)에는 커스텀 헤더가
// 실리지 않아서, 여기까지 내려오면 401 이 되고 본 요청은 아예 못 뜬다.
func Auth(token string, next http.Handler) http.Handler {
	if token == "" {
		return next
	}
	expected := []byte(token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if openPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}
		if subtle.ConstantTimeCompare([]byte(r.Header.Get(TokenHeader)), expected) != 1 {
			writeError(w, http.StatusUnauthorized, "암호가 필요합니다")
			return
		}
		next.ServeHTTP(w, r)
	})
}
