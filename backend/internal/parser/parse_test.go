package parser

import (
	"os"
	"strings"
	"testing"
)

func parseFixture(t *testing.T) *Result {
	t.Helper()
	f, err := os.Open("testdata/sample_masked.xls")
	if err != nil {
		t.Fatalf("픽스처 열기 실패: %v", err)
	}
	defer f.Close()

	res, err := Parse(f)
	if err != nil {
		t.Fatalf("Parse 실패: %v", err)
	}
	return res
}

func TestParseAccounts(t *testing.T) {
	res := parseFixture(t)

	if len(res.Accounts) != 3 {
		t.Fatalf("계좌 개수 = %d, want 3", len(res.Accounts))
	}

	want := []struct {
		number, display, accType string
		total, withdrawable      float64
	}{
		{"111111111110", "111-1111-1111-0", "ISA(중개형)", 1000000, 100000},
		{"222222222220", "222-2222-2222-0", "종합", 2500000, 250000},
		{"333333333330", "333-3333-3333-0", "연금저축계좌(신)", 3000000, 0},
	}
	for i, w := range want {
		got := res.Accounts[i]
		if got.Number != w.number || got.DisplayNumber != w.display || got.Type != w.accType {
			t.Errorf("Accounts[%d] = {%s %s %s}, want {%s %s %s}",
				i, got.Number, got.DisplayNumber, got.Type, w.number, w.display, w.accType)
		}
		if got.TotalAsset != w.total || got.Withdrawable != w.withdrawable {
			t.Errorf("Accounts[%d] 금액 = {%v %v}, want {%v %v}",
				i, got.TotalAsset, got.Withdrawable, w.total, w.withdrawable)
		}
	}
}

// 계좌현황은 "222-2222-2222-0", 상품보유현황 헤더는 "222-22-2222220" 으로
// 하이픈 위치가 다르다. 정규화 후 같은 키가 되어야 종목을 계좌에 붙일 수 있다.
func TestHoldingsJoinToAccount(t *testing.T) {
	res := parseFixture(t)

	const want = "222222222220"
	for _, h := range res.Holdings {
		if h.AccountNumber != want {
			t.Fatalf("%s 의 AccountNumber = %q, want %q", h.Name, h.AccountNumber, want)
		}
	}

	var matched bool
	for _, a := range res.Accounts {
		if a.Number == want {
			matched = true
		}
	}
	if !matched {
		t.Errorf("보유종목의 계좌번호 %q 와 일치하는 계좌가 없음", want)
	}
}

func TestParseHoldings(t *testing.T) {
	res := parseFixture(t)

	// 5행 중 미국달러는 Cash 로 빠지므로 종목은 4개다.
	if len(res.Holdings) != 4 {
		t.Fatalf("종목 개수 = %d, want 4", len(res.Holdings))
	}
	for _, h := range res.Holdings {
		if h.Name == "미국달러" {
			t.Error("미국달러는 Holdings 가 아니라 Cash 에 있어야 함")
		}
	}

	// 콤마·소수점·퍼센트가 섞인 일반 행
	h := res.Holdings[2]
	if h.Name != "KODEX 레버리지" {
		t.Fatalf("Holdings[2].Name = %q", h.Name)
	}
	if h.Quantity != 7 || h.EvalAmount != 86419.69 {
		t.Errorf("수량/평가금액 = %v/%v, want 7/86419.69", h.Quantity, h.EvalAmount)
	}
	if got := deref(t, h.ProfitRate); got != -17.70 {
		t.Errorf("손익률 = %v, want -17.70", got)
	}
	if got := deref(t, h.AvgBuyPrice); got != 15000 {
		t.Errorf("평균매입가 = %v, want 15000", got)
	}

	// HTML 엔티티가 상품명에 섞인 경우 (TIGER 미국S&P500)
	if res.Holdings[1].Name != "TIGER 미국S&P500" {
		t.Errorf("Holdings[1].Name = %q, want %q", res.Holdings[1].Name, "TIGER 미국S&P500")
	}
}

// 외화 마이너스 잔고 행: 수량·평가금액은 음수 값이고, 평단·손익·손익률만 "-"(nil) 이다.
// "-" 하나를 null 로 보되 "-84,045.00" 은 음수로 살려야 하는 게 이 파일의 함정이다.
// 종목이 아니므로 Holdings 가 아니라 Cash 로 빠진다.
func TestParseCashRow(t *testing.T) {
	res := parseFixture(t)

	if len(res.Cash) != 1 {
		t.Fatalf("Cash 개수 = %d, want 1", len(res.Cash))
	}
	h := res.Cash[0]
	if h.Name != "미국달러" {
		t.Fatalf("Cash[0].Name = %q, want 미국달러", h.Name)
	}
	if h.Quantity != -58.32 {
		t.Errorf("보유수량 = %v, want -58.32", h.Quantity)
	}
	if h.EvalAmount != -84045 {
		t.Errorf("평가금액 = %v, want -84045", h.EvalAmount)
	}
	if got := deref(t, h.BuyAmount); got != -84045 {
		t.Errorf("매입금액 = %v, want -84045", got)
	}
	if got := deref(t, h.CurrentPrice); got != 1441.10 {
		t.Errorf("현재가 = %v, want 1441.10", got)
	}
	if h.AvgBuyPrice != nil {
		t.Errorf("평균매입가 = %v, want nil", *h.AvgBuyPrice)
	}
	if h.ProfitLoss != nil {
		t.Errorf("평가손익 = %v, want nil", *h.ProfitLoss)
	}
	if h.ProfitRate != nil {
		t.Errorf("손익률 = %v, want nil", *h.ProfitRate)
	}
}

func TestParseRejectsNonBalanceFile(t *testing.T) {
	_, err := Parse(strings.NewReader("<html><body><p>다른 파일</p></body></html>"))
	if err == nil {
		t.Error("잔고파일이 아닌 입력에 에러를 기대했음")
	}
}

func deref(t *testing.T, p *float64) float64 {
	t.Helper()
	if p == nil {
		t.Fatal("값을 기대했는데 nil")
	}
	return *p
}
