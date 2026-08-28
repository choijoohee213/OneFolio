// Package api 는 잔고파일을 받아 자산배분 집계를 돌려주는 HTTP 계층이다.
// 받은 파일과 계산 결과는 응답 후 버린다. 어떤 자산 데이터도 서버에 남기지 않는다.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"io"

	"github.com/choijoohee213/OneFolio/backend/internal/classify"
	"github.com/choijoohee213/OneFolio/backend/internal/domain"
	"github.com/choijoohee213/OneFolio/backend/internal/master"
	"github.com/choijoohee213/OneFolio/backend/internal/ocr"
	"github.com/choijoohee213/OneFolio/backend/internal/parser"
	"github.com/choijoohee213/OneFolio/backend/internal/portfolio"
	"github.com/choijoohee213/OneFolio/backend/internal/quote"
)

const (
	maxUploadBytes = 32 << 20
	filesField     = "files"
	overridesField = "overrides"

	manualAccountsField = "manualAccounts"
	manualHoldingsField = "manualHoldings"
	holdingEditsField   = "holdingEdits"
	stockMappingsField  = "stockMappings"

	defaultSearchLimit = 20
)

type Server struct {
	listings    *master.Table
	ocrClient   *ocr.Client
	quoteClient *quote.Client
}

func New(listings *master.Table, ocrClient *ocr.Client, quoteClient *quote.Client) *Server {
	return &Server{listings: listings, ocrClient: ocrClient, quoteClient: quoteClient}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /api/stocks", s.searchStocks)
	mux.HandleFunc("POST /api/portfolio", s.portfolio)
	if s.ocrClient != nil {
		mux.HandleFunc("POST /api/ocr", s.extractFromScreenshot)
	}
	if s.quoteClient != nil {
		mux.HandleFunc("POST /api/quotes", s.quotes)
	}
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) searchStocks(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSON(w, http.StatusOK, []master.Entry{})
		return
	}
	limit := defaultSearchLimit
	results := s.listings.Search(query, limit)
	if results == nil {
		results = []master.Entry{}
	}
	writeJSON(w, http.StatusOK, results)
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

	stockMappings, err := parseStockMappings(r.FormValue(stockMappingsField))
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

	// stockMappings 로 지정된 종목은 마스터에서 종류를 찾아 분류에 반영한다.
	if len(stockMappings) > 0 {
		if overrides == nil {
			overrides = make(map[string]domain.Category)
		}
		for holdingName, code := range stockMappings {
			if _, ok := overrides[holdingName]; ok {
				continue
			}
			if entry, ok := s.listings.LookupByCode(code); ok {
				overrides[holdingName] = classify.FromKind(entry.Kind, holdingName)
			}
		}
	}

	classifier := classify.New(s.listings, overrides, stockMappings)
	summary := portfolio.Summarize(merged, classifier, s.listings, stockMappings)
	summary.Sources = sources
	writeJSON(w, http.StatusOK, summary)
}

var allowedImageTypes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/webp": true,
}

func (s *Server) extractFromScreenshot(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		writeError(w, http.StatusBadRequest, "업로드를 읽지 못했습니다: %v", err)
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "image 필드가 필요합니다")
		return
	}
	defer file.Close()

	mimeType := header.Header.Get("Content-Type")
	if !allowedImageTypes[mimeType] {
		writeError(w, http.StatusBadRequest, "지원하지 않는 이미지 형식입니다 (PNG, JPEG, WebP만 가능)")
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "이미지를 읽지 못했습니다")
		return
	}

	result, err := s.ocrClient.Extract(r.Context(), data, mimeType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "종목 추출 실패: %v", err)
		return
	}

	// 종목마다 currentPrice·avgBuyPrice 의 통화를 확정한다. Gemini 가 스스로
	// 판단한 currency 필드는 종목마다 빠뜨리기 쉬워서 믿을 수 없다 — 종목마스터로
	// 해외/국내가 확정되면 그걸 우선한다. 마스터에 없는 종목만 Gemini 판단을 쓴다.
	needsFx := false
	for i, h := range result.Holdings {
		// 티커를 먼저 본다. 같은 이름이 국내와 해외에 다 있을 때(SK하이닉스와
		// 그 ADR 처럼) 이름부터 맞추면 국내가 잡히고 티커는 버려진다.
		listing, ok := s.listings.Lookup(h.Name)
		if h.Ticker != "" {
			if entry, byTicker := s.listings.LookupByCode(h.Ticker); byTicker {
				result.Holdings[i].Name = entry.Name
				listing, ok = master.Listing{Code: entry.Code, Kind: entry.Kind, Market: entry.Market}, true
			}
		}
		if ok {
			if listing.Kind.IsForeign() {
				result.Holdings[i].Currency = "USD"
			} else {
				result.Holdings[i].Currency = "KRW"
			}
		}
		if result.Holdings[i].Currency == "USD" {
			needsFx = true
		}
	}

	// 화면에서 읽지 못한 값만 여기서 채운다. 통화가 종목마스터로 확정된 뒤라야
	// 달러를 원화로 옮길 수 있어 이 자리에서 한다.
	//
	// 화면에 있던 값은 건드리지 않는다. 예전에는 통화를 혼동했을까 봐 덮어썼는데,
	// 그 탓에 화면에 원화로 133,180원이라 적힌 종목이 우리 환율로 다시 계산되어
	// 131,095원으로 바뀌었다. 증권사가 계산해 둔 값이 더 믿을 만하다.
	var rate *float64
	if needsFx && s.quoteClient != nil {
		if v, err := s.quoteClient.ExchangeRate(r.Context(), "USD", "KRW"); err == nil {
			rate = &v
		}
	}
	ocr.FillCalculated(result.Holdings, rate)

	writeJSON(w, http.StatusOK, result)
}

type quoteResult struct {
	Price     float64 `json:"price"`
	Currency  string  `json:"currency"`
	PrevClose float64 `json:"prevClose,omitempty"`
}

type quotesResponse struct {
	Quotes map[string]quoteResult `json:"quotes"`
	// UsdKrw 은 조회한 종목 중 원화가 아닌 게 있을 때만 채운다.
	UsdKrw float64 `json:"usdKrw,omitempty"`
}

// resolveSymbols 는 종목코드가 국내인지 해외인지 종목마스터로 판별한다.
// 코드 모양으로는 가를 수 없다 — 국내에도 0167A0 같은 영문 섞인 코드와
// Q580074 같은 ETN 코드가 있어서, 숫자 여부로 판단하면 국내 종목의 29%가
// 해외로 잘못 분류돼 조회에 실패한다.
// 마스터에 없는 코드(프론트에 남은 옛 데이터 등)만 첫 글자로 짐작한다.
func (s *Server) resolveSymbols(codes []string) []quote.Symbol {
	symbols := make([]quote.Symbol, 0, len(codes))
	for _, code := range codes {
		foreign := code != "" && (code[0] < '0' || code[0] > '9')
		if entry, ok := s.listings.LookupByCode(code); ok {
			foreign = entry.Kind.IsForeign()
		}
		symbols = append(symbols, quote.Symbol{Code: code, Foreign: foreign})
	}
	return symbols
}

// quotes 는 이미 알고 있는 종목(파일로 올렸거나 수동으로 넣은)의 현재가만
// 새로고침한다. 수량·평단가는 그대로 두고 프론트에서 evalAmount 를 다시 낸다.
func (s *Server) quotes(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Codes []string `json:"codes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "요청을 읽지 못했습니다: %v", err)
		return
	}
	if len(body.Codes) == 0 {
		writeError(w, http.StatusBadRequest, "codes 가 비어 있습니다")
		return
	}

	prices, err := s.quoteClient.Prices(r.Context(), s.resolveSymbols(body.Codes))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "시세 조회 실패: %v", err)
		return
	}

	resp := quotesResponse{Quotes: make(map[string]quoteResult, len(prices))}
	hasForeign := false
	for code, p := range prices {
		resp.Quotes[code] = quoteResult{Price: p.Price, Currency: p.Currency, PrevClose: p.PrevClose}
		if p.Currency != "KRW" {
			hasForeign = true
		}
	}

	if hasForeign {
		rate, err := s.quoteClient.ExchangeRate(r.Context(), "USD", "KRW")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "환율 조회 실패: %v", err)
			return
		}
		resp.UsdKrw = rate
	}

	writeJSON(w, http.StatusOK, resp)
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
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		EvalAmount  float64  `json:"evalAmount"`
		AccountID   string   `json:"accountId"`
		Quantity    *float64 `json:"quantity"`
		AvgBuyPrice *float64 `json:"avgBuyPrice"`
		BuyAmount   *float64 `json:"buyAmount"`
		ProfitLoss  *float64 `json:"profitLoss"`
		ProfitRate  *float64 `json:"profitRate"`
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

		var quantity float64
		if input.Quantity != nil {
			quantity = *input.Quantity
		}

		// 매입금액을 받았으면 그걸 쓴다. 평단은 화면에서 반올림된 값이라
		// 수량을 곱하면 원 단위가 어긋난다(예: 271,222×9=2,440,998, 실제 2,441,000).
		buyAmount := input.BuyAmount
		if buyAmount == nil && input.AvgBuyPrice != nil && quantity > 0 {
			v := *input.AvgBuyPrice * quantity
			buyAmount = &v
		}

		holdings = append(holdings, domain.Holding{
			AccountNumber: accountNumber,
			Name:          name,
			Quantity:      quantity,
			AvgBuyPrice:   input.AvgBuyPrice,
			BuyAmount:     buyAmount,
			EvalAmount:    input.EvalAmount,
			ProfitLoss:    input.ProfitLoss,
			ProfitRate:    input.ProfitRate,
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

func parseStockMappings(raw string) (map[string]string, error) {
	if raw == "" {
		return nil, nil
	}
	var mappings map[string]string
	if err := json.Unmarshal([]byte(raw), &mappings); err != nil {
		return nil, fmt.Errorf("%s 는 JSON 객체여야 합니다", stockMappingsField)
	}
	return mappings, nil
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
		// 분류 이름이 바뀌기 전에 저장해 둔 값이 올라올 수 있다. 그대로 물리면
		// 화면 전체가 400 으로 막히므로 지금 이름으로 옮겨서 받는다.
		migrated := category.Migrate()
		if !migrated.Valid() {
			return nil, fmt.Errorf("%s: 알 수 없는 카테고리 %q", name, category)
		}
		overrides[name] = migrated
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
