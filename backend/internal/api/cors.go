package api

import (
	"net/http"
	"strings"
)

// CORS 는 허용 오리진을 붙인다. 프론트가 다른 도메인(Cloudflare Pages)에 있어서 필요하다.
// origins 가 비면 모두 허용한다. 서버가 아무 데이터도 저장하지 않고 사용자가 직접
// 올린 파일만 계산해서 돌려주므로 기본값을 열어둔다.
func CORS(origins []string, next http.Handler) http.Handler {
	allowed := make(map[string]bool, len(origins))
	for _, origin := range origins {
		if trimmed := strings.TrimSpace(origin); trimmed != "" {
			allowed[trimmed] = true
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := allowFor(allowed, r.Header.Get("Origin")); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func allowFor(allowed map[string]bool, origin string) string {
	if len(allowed) == 0 {
		return "*"
	}
	if origin != "" && allowed[origin] {
		return origin
	}
	return ""
}
