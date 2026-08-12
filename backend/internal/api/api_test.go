package api

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/choijoohee213/OneFolio/backend/internal/master"
	"github.com/choijoohee213/OneFolio/backend/internal/portfolio"
)

const fixture = "../parser/testdata/sample_masked.xls"

func newServer(t *testing.T) http.Handler {
	t.Helper()
	listings, err := master.Load()
	if err != nil {
		t.Fatalf("종목마스터 로드 실패: %v", err)
	}
	mux := http.NewServeMux()
	New(listings).Register(mux)
	return mux
}

func uploadRequest(t *testing.T, overrides string, fileCount int) *http.Request {
	return buildRequest(t, overrides, fileCount, "")
}

func buildRequest(t *testing.T, overrides string, fileCount int, manualHoldings string) *http.Request {
	return buildFullRequest(t, overrides, fileCount, "", manualHoldings)
}

func buildFullRequest(t *testing.T, overrides string, fileCount int, manualAccounts, manualHoldings string) *http.Request {
	t.Helper()
	content, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("픽스처 읽기 실패: %v", err)
	}

	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	for i := 0; i < fileCount; i++ {
		part, err := form.CreateFormFile(filesField, "잔고.xls")
		if err != nil {
			t.Fatal(err)
		}
		part.Write(content)
	}
	if overrides != "" {
		form.WriteField(overridesField, overrides)
	}
	if manualAccounts != "" {
		form.WriteField(manualAccountsField, manualAccounts)
	}
	if manualHoldings != "" {
		form.WriteField(manualHoldingsField, manualHoldings)
	}
	form.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/portfolio", &body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	return req
}

func decodeSummary(t *testing.T, body io.Reader) portfolio.Summary {
	t.Helper()
	var summary portfolio.Summary
	if err := json.NewDecoder(body).Decode(&summary); err != nil {
		t.Fatalf("응답 디코딩 실패: %v", err)
	}
	return summary
}

func TestHealth(t *testing.T) {
	rec := httptest.NewRecorder()
	newServer(t).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestPortfolioUpload(t *testing.T) {
	rec := httptest.NewRecorder()
	newServer(t).ServeHTTP(rec, uploadRequest(t, "", 1))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}

	summary := decodeSummary(t, rec.Body)
	if summary.TotalAsset != 6500000 {
		t.Errorf("totalAsset = %v, want 6500000", summary.TotalAsset)
	}
	if len(summary.Holdings) != 4 {
		t.Errorf("종목 %d개, want 4", len(summary.Holdings))
	}
	if len(summary.Categories) == 0 {
		t.Error("categories 가 비어있음")
	}
}

// 같은 파일을 두 번 올려도 종목이 중복되지 않아야 한다.
func TestPortfolioDeduplicatesUploads(t *testing.T) {
	rec := httptest.NewRecorder()
	newServer(t).ServeHTTP(rec, uploadRequest(t, "", 2))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	if summary := decodeSummary(t, rec.Body); len(summary.Holdings) != 4 {
		t.Errorf("종목 %d개, want 4", len(summary.Holdings))
	}
}

func TestPortfolioAppliesOverrides(t *testing.T) {
	rec := httptest.NewRecorder()
	newServer(t).ServeHTTP(rec, uploadRequest(t, `{"삼성전자":"현금성"}`, 1))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	for _, holding := range decodeSummary(t, rec.Body).Holdings {
		if holding.Name == "삼성전자" && holding.Category != "현금성" {
			t.Errorf("삼성전자 category = %q, want 현금성", holding.Category)
		}
	}
}

// 파일 없이 직접 추가한 자산만 보내도 계산이 되어야 한다.
func TestPortfolioAcceptsManualHoldingsWithoutFiles(t *testing.T) {
	rec := httptest.NewRecorder()
	newServer(t).ServeHTTP(rec, buildRequest(t, "", 0,
		`[{"id":"a1","name":"예금","evalAmount":1000000}]`))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	summary := decodeSummary(t, rec.Body)
	if summary.CoveredAsset != 1000000 {
		t.Errorf("coveredAsset = %v, want 1000000", summary.CoveredAsset)
	}
	if len(summary.Holdings) != 1 || summary.Holdings[0].Name != "예금" {
		t.Errorf("holdings = %+v, want [예금]", summary.Holdings)
	}
}

// 파일과 직접 추가한 자산을 함께 보내면 둘 다 반영되어야 한다.
func TestPortfolioMergesManualHoldingsWithFiles(t *testing.T) {
	rec := httptest.NewRecorder()
	newServer(t).ServeHTTP(rec, buildRequest(t, "", 1,
		`[{"id":"a1","name":"예금","evalAmount":1000000}]`))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	summary := decodeSummary(t, rec.Body)
	// 픽스처는 계좌 3개 요약에 종목 상세는 한 계좌(2,500,000)만 담고 있다.
	if want := 2500000.0 + 1000000; summary.CoveredAsset != want {
		t.Errorf("coveredAsset = %v, want %v", summary.CoveredAsset, want)
	}
	if len(summary.Holdings) != 5 {
		t.Errorf("종목 %d개, want 5 (파일 4 + 직접 추가 1)", len(summary.Holdings))
	}
}

// 잔고파일도 없이 직접 만든 계좌만으로도 계산이 되어야 한다.
func TestPortfolioAcceptsManualAccountWithoutFiles(t *testing.T) {
	rec := httptest.NewRecorder()
	newServer(t).ServeHTTP(rec, buildFullRequest(t, "", 0,
		`[{"id":"acc1","name":"저축은행","totalAsset":8000000}]`, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	summary := decodeSummary(t, rec.Body)
	if summary.CoveredAsset != 8000000 {
		t.Errorf("coveredAsset = %v, want 8000000", summary.CoveredAsset)
	}
	if len(summary.Accounts) != 1 || !summary.Accounts[0].Covered {
		t.Errorf("accounts = %+v, want 집계된 계좌 1개", summary.Accounts)
	}
}

// accountId 를 준 종목은 그 계좌에 붙어야 하고, 계좌 총액에서 자기 몫을 뗀
// 나머지만 현금성으로 남아야 한다.
func TestPortfolioAttachesManualHoldingToManualAccount(t *testing.T) {
	rec := httptest.NewRecorder()
	newServer(t).ServeHTTP(rec, buildFullRequest(t, "", 0,
		`[{"id":"acc1","name":"저축은행","totalAsset":8000000}]`,
		`[{"id":"h1","name":"정기예금","evalAmount":5000000,"accountId":"acc1"}]`))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	summary := decodeSummary(t, rec.Body)
	if summary.CoveredAsset != 8000000 {
		t.Errorf("coveredAsset = %v, want 8000000 (계좌 총액만, 종목분을 또 더하면 안 됨)", summary.CoveredAsset)
	}
}

// 계좌만 있고 종목이 하나도 없으면 Go 의 nil 슬라이스가 JSON null 로 새어나가기
// 쉽다. decodeSummary 는 null 을 nil 슬라이스로 조용히 받아버려 이 버그를 못
// 잡으므로, 응답 본문 문자열에서 직접 확인한다.
func TestPortfolioHoldingsIsEmptyArrayNotNull(t *testing.T) {
	rec := httptest.NewRecorder()
	newServer(t).ServeHTTP(rec, buildFullRequest(t, "", 0,
		`[{"id":"acc1","name":"저축은행","totalAsset":8000000}]`, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("응답 디코딩 실패: %v", err)
	}
	for _, field := range []string{"holdings", "categories"} {
		if got := string(raw[field]); got == "null" {
			t.Errorf("%s = null, want [] (프런트가 .map 을 못 돌려 화면이 죽는다)", field)
		}
	}
}

// 계좌번호를 적어 직접 만든 계좌가 있는데 나중에 그 계좌의 잔고파일을 올리면,
// 파일이 실제 총액과 종목을 갖고 있으니 파일이 이겨야 한다. 수동 계좌가 남으면
// 같은 계좌가 두 줄이 되어 자산이 두 번 세어진다.
func TestPortfolioFileSupersedesManualAccountWithSameNumber(t *testing.T) {
	// 픽스처의 종합 계좌는 222-2222-2222-0 (정규화 222222222220) 이다.
	rec := httptest.NewRecorder()
	newServer(t).ServeHTTP(rec, buildFullRequest(t, "", 1,
		`[{"id":"acc1","name":"내가 적은 종합계좌","totalAsset":9999999,"accountNumber":"222-2222-2222-0"}]`, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	summary := decodeSummary(t, rec.Body)

	for _, account := range summary.Accounts {
		if account.Type == "내가 적은 종합계좌" {
			t.Errorf("잔고파일이 같은 계좌를 담고 있으면 수동 계좌는 빠져야 한다: %+v", summary.Accounts)
		}
	}
	if summary.CoveredAsset != 2500000 {
		t.Errorf("coveredAsset = %v, want 2500000 (수동 계좌 총액이 섞이면 안 된다)", summary.CoveredAsset)
	}
}

// 계좌번호가 파일의 어느 계좌와도 안 겹치면 수동 계좌는 그대로 살아 있어야 한다.
func TestPortfolioKeepsManualAccountWithUnrelatedNumber(t *testing.T) {
	rec := httptest.NewRecorder()
	newServer(t).ServeHTTP(rec, buildFullRequest(t, "", 1,
		`[{"id":"acc1","name":"저축은행","totalAsset":8000000,"accountNumber":"999-9999-9999-9"}]`, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	summary := decodeSummary(t, rec.Body)

	var found bool
	for _, account := range summary.Accounts {
		if account.Type == "저축은행" {
			found = true
		}
	}
	if !found {
		t.Errorf("겹치지 않는 수동 계좌는 남아야 한다: %+v", summary.Accounts)
	}
	if want := 2500000.0 + 8000000; summary.CoveredAsset != want {
		t.Errorf("coveredAsset = %v, want %v", summary.CoveredAsset, want)
	}
}

func TestPortfolioRejectsBadRequests(t *testing.T) {
	server := newServer(t)

	tests := []struct {
		name    string
		request *http.Request
		want    int
	}{
		{"파일도 직접 추가한 자산도 없음", uploadRequest(t, "", 0), http.StatusBadRequest},
		{"잘못된 카테고리", uploadRequest(t, `{"삼성전자":"없는분류"}`, 1), http.StatusBadRequest},
		{"JSON 아닌 overrides", uploadRequest(t, `삼성전자=현금성`, 1), http.StatusBadRequest},
		{"직접 추가한 자산의 평가금액이 0 이하", buildRequest(t, "", 0, `[{"id":"a1","name":"예금","evalAmount":0}]`), http.StatusBadRequest},
		{"직접 추가한 자산의 이름이 비어있음", buildRequest(t, "", 0, `[{"id":"a1","name":"","evalAmount":1000}]`), http.StatusBadRequest},
		{"직접 추가한 계좌의 총자산이 0 이하", buildFullRequest(t, "", 0, `[{"id":"acc1","name":"저축은행","totalAsset":0}]`, ""), http.StatusBadRequest},
		{"직접 추가한 계좌의 이름이 비어있음", buildFullRequest(t, "", 0, `[{"id":"acc1","name":"","totalAsset":1000}]`, ""), http.StatusBadRequest},
		{
			"존재하지 않는 계좌를 가리키는 종목",
			buildFullRequest(t, "", 0, "", `[{"id":"h1","name":"정기예금","evalAmount":1000,"accountId":"없는id"}]`),
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, tt.request)
		if rec.Code != tt.want {
			t.Errorf("%s: status = %d, want %d (%s)", tt.name, rec.Code, tt.want, rec.Body)
		}
	}
}

// 잔고파일이 아닌 입력은 422 로 돌려준다.
func TestPortfolioRejectsNonBalanceFile(t *testing.T) {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, _ := form.CreateFormFile(filesField, "아무거나.xls")
	part.Write([]byte("<html><body><p>다른 파일</p></body></html>"))
	form.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/portfolio", &body)
	req.Header.Set("Content-Type", form.FormDataContentType())

	rec := httptest.NewRecorder()
	newServer(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
}

func TestCORSPreflight(t *testing.T) {
	handler := CORS([]string{"https://onefolio.pages.dev"}, newServer(t))

	req := httptest.NewRequest(http.MethodOptions, "/api/portfolio", nil)
	req.Header.Set("Origin", "https://onefolio.pages.dev")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://onefolio.pages.dev" {
		t.Errorf("Allow-Origin = %q", got)
	}
}

func TestCORSRejectsUnknownOrigin(t *testing.T) {
	handler := CORS([]string{"https://onefolio.pages.dev"}, newServer(t))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin = %q, want 빈 값", got)
	}
}
