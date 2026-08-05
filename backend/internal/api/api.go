// Package api 는 잔고파일을 받아 자산배분 집계를 돌려주는 HTTP 계층이다.
// 받은 파일과 계산 결과는 응답 후 버린다. 어떤 자산 데이터도 서버에 남기지 않는다.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/choijoohee213/OneFolio/backend/internal/classify"
	"github.com/choijoohee213/OneFolio/backend/internal/domain"
	"github.com/choijoohee213/OneFolio/backend/internal/master"
	"github.com/choijoohee213/OneFolio/backend/internal/parser"
	"github.com/choijoohee213/OneFolio/backend/internal/portfolio"
)

const (
	maxUploadBytes = 32 << 20
	filesField     = "files"
	overridesField = "overrides"
)

type Server struct {
	listings master.Table
}

func New(listings master.Table) *Server {
	return &Server{listings: listings}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("POST /api/portfolio", s.portfolio)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) portfolio(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		writeError(w, http.StatusBadRequest, "업로드를 읽지 못했습니다: %v", err)
		return
	}
	defer r.MultipartForm.RemoveAll()

	uploads := r.MultipartForm.File[filesField]
	if len(uploads) == 0 {
		writeError(w, http.StatusBadRequest, "%s 필드에 잔고파일이 없습니다", filesField)
		return
	}

	overrides, err := parseOverrides(r.FormValue(overridesField))
	if err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}

	results := make([]*parser.Result, 0, len(uploads))
	sources := make([]portfolio.Source, 0, len(uploads))
	for _, upload := range uploads {
		file, err := upload.Open()
		if err != nil {
			writeError(w, http.StatusBadRequest, "%s 를 열지 못했습니다", upload.Filename)
			return
		}
		result, err := parser.Parse(file)
		file.Close()
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "%s 파싱 실패: %v", upload.Filename, err)
			return
		}
		results = append(results, result)
		sources = append(sources, portfolio.Source{
			FileName:       upload.Filename,
			AccountNumbers: portfolio.CoveredAccounts(result),
		})
	}

	classifier := classify.New(s.listings, overrides)
	summary := portfolio.Summarize(portfolio.Merge(results...), classifier)
	summary.Sources = sources
	writeJSON(w, http.StatusOK, summary)
}

func parseOverrides(raw string) (map[string]domain.Category, error) {
	if raw == "" {
		return nil, nil
	}
	var overrides map[string]domain.Category
	if err := json.Unmarshal([]byte(raw), &overrides); err != nil {
		return nil, fmt.Errorf("%s 는 JSON 객체여야 합니다", overridesField)
	}
	for name, category := range overrides {
		if !category.Valid() {
			return nil, fmt.Errorf("%s: 알 수 없는 카테고리 %q", name, category)
		}
	}
	return overrides, nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, format string, args ...any) {
	writeJSON(w, status, map[string]string{"error": fmt.Sprintf(format, args...)})
}
