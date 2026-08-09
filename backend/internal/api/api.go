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
	holdingEditsField   = "holdingEdits"
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

	holdingEdits, err := parseHoldingEdits(r.FormValue(holdingEditsField))
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

	merged := portfolio.Merge(results...)

	// 파일 계좌번호 + 수동 계좌 id 를 모두 유효한 소속 계좌로 본다.
	// 수동 종목을 파일 계좌에 붙일 수 있어야 하기 때문이다.
	validAccounts := make(map[string]string, len(merged.Accounts)+len(manualAccounts))
	for _, account := range merged.Accounts {
		validAccounts[account.Number] = account.Number
	}
	for _, account := range manualAccounts {
		validAccounts[account.id] = portfolio.ManualAccountPrefix + account.id
	}

	manualHoldings, err := parseManualHoldings(r.FormValue(manualHoldingsField), validAccounts)
	if err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}

	if len(uploads) == 0 && len(manualAccounts) == 0 && len(manualHoldings) == 0 {
		writeError(w, http.StatusBadRequest, "%s 필드에 잔고파일이 없습니다", filesField)
		return
	}

	// 직접 만든 계좌에 계좌번호를 적어 뒀는데 그 계좌의 잔고파일이 올라왔다면,
	// 파일 쪽이 실제 총액과 종목 상세를 갖고 있으니 파일이 이긴다. 수동 계좌를
	// 그대로 두면 같은 계좌가 두 줄로 잡혀 자산이 두 번 세어진다.
	fromFile := make(map[string]bool, len(merged.Accounts))
	for _, account := range merged.Accounts {
		fromFile[account.Number] = true
	}
	superseded := make(map[string]bool)
	for _, manual := range manualAccounts {
		if manual.realNumber != "" && fromFile[manual.realNumber] {
			superseded[manual.id] = true
			continue
		}
		merged.Accounts = append(merged.Accounts, manual.account)
	}

	for _, holding := range manualHoldings {
		// 대체된 계좌에 붙어 있던 종목은 파일의 실제 종목으로 갈음된다.
		if id, ok := manualAccountIDOf(holding.AccountNumber); ok && superseded[id] {
			continue
		}
		merged.Holdings = append(merged.Holdings, holding)
	}

	portfolio.ApplyEdits(merged, holdingEdits)

	classifier := classify.New(s.listings, overrides)
	summary := portfolio.Summarize(merged, classifier)
	summary.Sources = sources
	writeJSON(w, http.StatusOK, summary)
}

func manualAccountIDOf(accountNumber string) (string, bool) {
	if !portfolio.IsManualAccount(accountNumber) {
		return "", false
	}
	return strings.TrimPrefix(accountNumber, portfolio.ManualAccountPrefix), true
}

// manualAccount 는 직접 만든 계좌 하나다. Account.Number 에는 항상
// ManualAccountPrefix 를 붙여 파일 계좌와 구분하고, 사용자가 적어 넣은
// 실제 계좌번호는 RealNumber 에 따로 둔다 — 나중에 같은 계좌의 잔고파일이
// 올라왔는지 맞춰보는 데에만 쓴다.
type manualAccount struct {
	id         string
	realNumber string
	account    domain.Account
}

// parseManualAccounts 는 잔고파일 없이 사용자가 이름과 총액만으로 직접 만든
// 계좌를 읽는다. 진짜 계좌와 똑같이 취급되므로 상세 종목이 없어도 된다.
func parseManualAccounts(raw string) ([]manualAccount, error) {
	if raw == "" {
		return nil, nil
	}

	var inputs []struct {
		ID            string  `json:"id"`
		Name          string  `json:"name"`
		TotalAsset    float64 `json:"totalAsset"`
		AccountNumber string  `json:"accountNumber"`
	}
	if err := json.Unmarshal([]byte(raw), &inputs); err != nil {
		return nil, fmt.Errorf("%s 는 JSON 배열이어야 합니다", manualAccountsField)
	}

	accounts := make([]manualAccount, 0, len(inputs))
	for _, input := range inputs {
		id := strings.TrimSpace(input.ID)
		name := strings.TrimSpace(input.Name)
		if id == "" || name == "" {
			return nil, fmt.Errorf("%s: id 와 이름은 비어 있을 수 없습니다", manualAccountsField)
		}
		if input.TotalAsset <= 0 {
			return nil, fmt.Errorf("%s: 총자산은 0보다 커야 합니다", name)
		}
		accounts = append(accounts, manualAccount{
			id:         id,
			realNumber: domain.NormalizeAccountNumber(input.AccountNumber),
			account: domain.Account{
				Number:        portfolio.ManualAccountPrefix + id,
				DisplayNumber: strings.TrimSpace(input.AccountNumber),
				Type:          name,
				TotalAsset:    input.TotalAsset,
			},
		})
	}
	return accounts, nil
}

// parseManualHoldings 는 잔고파일에 없는, 사용자가 직접 추가한 종목을 읽는다.
// 종목명으로 분류하는 overrides 와 달리 이건 종목 자체가 어디에도 없으니
// 프런트가 준 id 로 구분한다. accountID 를 주면 직접 추가한 계좌에 붙고,
// 비우면 어느 계좌에도 안 속한 채 자기 몫만 집계에 잡힌다.
// validAccounts: accountId → 실제 accountNumber 매핑.
// 수동 계좌는 id → "manual-account:id", 파일 계좌는 번호 → 번호.
func parseManualHoldings(raw string, validAccounts map[string]string) ([]domain.Holding, error) {
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
			mapped, ok := validAccounts[accountID]
			if !ok {
				return nil, fmt.Errorf("%s: 알 수 없는 계좌입니다", name)
			}
			accountNumber = mapped
		}

		holdings = append(holdings, domain.Holding{
			AccountNumber: accountNumber,
			Name:          name,
			EvalAmount:    input.EvalAmount,
		})
	}
	return holdings, nil
}

// parseHoldingEdits 는 잔고파일 종목의 값을 직접 고친 내용을 읽는다.
// 계좌번호+종목명으로 대상을 찾으므로 어느 계좌 것인지가 반드시 있어야 한다.
func parseHoldingEdits(raw string) ([]portfolio.HoldingEdit, error) {
	if raw == "" {
		return nil, nil
	}

	var edits []portfolio.HoldingEdit
	if err := json.Unmarshal([]byte(raw), &edits); err != nil {
		return nil, fmt.Errorf("%s 는 JSON 배열이어야 합니다", holdingEditsField)
	}

	for i := range edits {
		edits[i].AccountNumber = strings.TrimSpace(edits[i].AccountNumber)
		edits[i].Name = strings.TrimSpace(edits[i].Name)
		if edits[i].AccountNumber == "" || edits[i].Name == "" {
			return nil, fmt.Errorf("%s: 계좌번호와 종목명은 비어 있을 수 없습니다", holdingEditsField)
		}
		if edits[i].Quantity != nil && *edits[i].Quantity < 0 {
			return nil, fmt.Errorf("%s: 보유수량은 0보다 작을 수 없습니다", edits[i].Name)
		}
	}
	return edits, nil
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
