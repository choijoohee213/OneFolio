// Package api 는 잔고파일을 받아 자산배분 집계를 돌려주는 HTTP 계층이다.
// 받은 파일과 계산 결과는 응답 후 버린다. 어떤 자산 데이터도 서버에 남기지 않는다.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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

	manualHoldingsField = "manualHoldings"
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

	overrides, err := parseOverrides(r.FormValue(overridesField))
	if err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}

	manualHoldings, err := parseManualHoldings(r.FormValue(manualHoldingsField))
	if err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}

	if len(uploads) == 0 && len(manualHoldings) == 0 {
		writeError(w, http.StatusBadRequest, "%s 필드에 잔고파일이 없습니다", filesField)
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

	merged := portfolio.Merge(results...)
	merged.Holdings = append(merged.Holdings, manualHoldings...)

	classifier := classify.New(s.listings, overrides)
	summary := portfolio.Summarize(merged, classifier)
	summary.Sources = sources
	writeJSON(w, http.StatusOK, summary)
}

// parseManualHoldings 는 잔고파일에 없는, 사용자가 직접 추가한 자산을 읽는다.
// 종목명으로 분류하는 overrides 와 달리 이건 종목 자체가 어디에도 없으니
// 프런트가 준 id 로 구분한다.
func parseManualHoldings(raw string) ([]domain.Holding, error) {
	if raw == "" {
		return nil, nil
	}

	var inputs []struct {
		ID         string  `json:"id"`
		Name       string  `json:"name"`
		EvalAmount float64 `json:"evalAmount"`
	}
	if err := json.Unmarshal([]byte(raw), &inputs); err != nil {
		return nil, fmt.Errorf("%s 는 JSON 배열이어야 합니다", manualHoldingsField)
	}

	holdings := make([]domain.Holding, 0, len(inputs))
	for _, input := range inputs {
		id := strings.TrimSpace(input.ID)
		name := strings.TrimSpace(input.Name)
		if id == "" || name == "" {
			return nil, fmt.Errorf("%s: id 와 종목명은 비어 있을 수 없습니다", manualHoldingsField)
		}
		if input.EvalAmount <= 0 {
			return nil, fmt.Errorf("%s: 평가금액은 0보다 커야 합니다", name)
		}
		holdings = append(holdings, domain.Holding{
			AccountNumber: portfolio.ManualAccountPrefix + id,
			Name:          name,
			EvalAmount:    input.EvalAmount,
		})
	}
	return holdings, nil
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
