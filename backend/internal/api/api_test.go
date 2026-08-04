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

func TestPortfolioRejectsBadRequests(t *testing.T) {
	server := newServer(t)

	tests := []struct {
		name    string
		request *http.Request
		want    int
	}{
		{"파일 없음", uploadRequest(t, "", 0), http.StatusBadRequest},
		{"잘못된 카테고리", uploadRequest(t, `{"삼성전자":"없는분류"}`, 1), http.StatusBadRequest},
		{"JSON 아닌 overrides", uploadRequest(t, `삼성전자=현금성`, 1), http.StatusBadRequest},
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
