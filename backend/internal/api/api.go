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

	manualAccountsField = "manualAccounts"
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

	manualAccounts, err := parseManualAccounts(r.FormValue(manualAccountsField))
	if err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}

	manualAccountIDs := make(map[string]bool, len(manualAccounts))
	for _, account := range manualAccounts {
		manualAccountIDs[strings.TrimPrefix(account.Number, portfolio.ManualAccountPrefix)] = true
	}

	manualHoldings, err := parseManualHoldings(r.FormValue(manualHoldingsField), manualAccountIDs)
	if err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}

	if len(uploads) == 0 && len(manualAccounts) == 0 && len(manualHoldings) == 0 {
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
	merged.Accounts = append(merged.Accounts, manualAccounts...)
	merged.Holdings = append(merged.Holdings, manualHoldings...)

	classifier := classify.New(s.listings, overrides)
	summary := portfolio.Summarize(merged, classifier)
	summary.Sources = sources
	writeJSON(w, http.StatusOK, summary)
}

// parseManualAccounts 는 잔고파일 없이 사용자가 이름과 총액만으로 직접 만든
// 계좌를 읽는다. 진짜 계좌와 똑같이 취급되므로 상세 종목이 없어도 된다.
func parseManualAccounts(raw string) ([]domain.Account, error) {
	if raw == "" {
		return nil, nil
	}

	var inputs []struct {
		ID         string  `json:"id"`
		Name       string  `json:"name"`
		TotalAsset float64 `json:"totalAsset"`
	}
	if err := json.Unmarshal([]byte(raw), &inputs); err != nil {
		return nil, fmt.Errorf("%s 는 JSON 배열이어야 합니다", manualAccountsField)
	}

	accounts := make([]domain.Account, 0, len(inputs))
	for _, input := range inputs {
		id := strings.TrimSpace(input.ID)
		name := strings.TrimSpace(input.Name)
		if id == "" || name == "" {
			return nil, fmt.Errorf("%s: id 와 이름은 비어 있을 수 없습니다", manualAccountsField)
		}
		if input.TotalAsset <= 0 {
			return nil, fmt.Errorf("%s: 총자산은 0보다 커야 합니다", name)
		}
		accounts = append(accounts, domain.Account{
			Number:     portfolio.ManualAccountPrefix + id,
			Type:       name,
			TotalAsset: input.TotalAsset,
		})
	}
	return accounts, nil
}

// parseManualHoldings 는 잔고파일에 없는, 사용자가 직접 추가한 종목을 읽는다.
// 종목명으로 분류하는 overrides 와 달리 이건 종목 자체가 어디에도 없으니
// 프런트가 준 id 로 구분한다. accountID 를 주면 직접 추가한 계좌에 붙고,
// 비우면 어느 계좌에도 안 속한 채 자기 몫만 집계에 잡힌다.
func parseManualHoldings(raw string, manualAccountIDs map[string]bool) ([]domain.Holding, error) {
	if raw == "" {
		return nil, nil
	}

	var inputs []struct {
		ID         string  `json:"id"`
		Name       string  `json:"name"`
		EvalAmount float64 `json:"evalAmount"`
		AccountID  string  `json:"accountId"`
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

		accountNumber := portfolio.ManualHoldingPrefix + id
		if accountID := strings.TrimSpace(input.AccountID); accountID != "" {
			if !manualAccountIDs[accountID] {
				return nil, fmt.Errorf("%s: 알 수 없는 계좌입니다", name)
			}
			accountNumber = portfolio.ManualAccountPrefix + accountID
		}

		holdings = append(holdings, domain.Holding{
			AccountNumber: accountNumber,
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
