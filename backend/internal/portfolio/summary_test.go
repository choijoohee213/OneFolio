package portfolio

import (
	"testing"

	"github.com/choijoohee213/OneFolio/backend/internal/classify"
	"github.com/choijoohee213/OneFolio/backend/internal/domain"
)

func priced(accountNo, name string, price, evalAmount float64) domain.Holding {
	h := holding(accountNo, name, evalAmount)
	h.CurrentPrice = &price
	return h
}

func categoryAmount(s Summary, c domain.Category) (float64, bool) {
	for _, total := range s.Categories {
		if total.Category == c {
			return total.Amount, true
		}
	}
	return 0, false
}

func testPortfolio() *Portfolio {
	return &Portfolio{
		Accounts: []domain.Account{account("111", 6000), account("222", 4000)},
		Holdings: []domain.Holding{
			priced("111", "삼성전자", 262500, 5000),
			priced("111", "TIGER 미국S&P500", 26545, 2000),
			priced("222", "AMD", 686179.76, 1000),
		},
	}
}

func TestSummarizeWeights(t *testing.T) {
	summary := Summarize(testPortfolio(), classify.New(nil))

	if summary.TotalAsset != 10000 {
		t.Fatalf("TotalAsset = %v, want 10000", summary.TotalAsset)
	}

	if amount, _ := categoryAmount(summary, domain.DomesticStock); amount != 5000 {
		t.Errorf("개별주(국내) = %v, want 5000", amount)
	}
	if amount, _ := categoryAmount(summary, domain.IndexETF); amount != 2000 {
		t.Errorf("지수 ETF = %v, want 2000", amount)
	}
	if amount, _ := categoryAmount(summary, domain.ForeignStock); amount != 1000 {
		t.Errorf("개별주(해외) = %v, want 1000", amount)
	}
}

// 종목으로 잡히지 않는 잔액(예수금 등)은 현금성으로 떨어져야 한다.
func TestSummarizeCashRemainder(t *testing.T) {
	summary := Summarize(testPortfolio(), classify.New(nil))

	amount, ok := categoryAmount(summary, domain.Cash)
	if !ok {
		t.Fatal("현금성 카테고리가 없음")
	}
	if amount != 2000 {
		t.Errorf("현금성 = %v, want 2000 (10000 - 8000)", amount)
	}
}

func TestSummarizeWeightsSumTo100(t *testing.T) {
	summary := Summarize(testPortfolio(), classify.New(nil))

	var sum float64
	for _, total := range summary.Categories {
		sum += total.Weight
	}
	if diff := sum - 100; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("카테고리 비중 합 = %v, want 100", sum)
	}
}

func TestSummarizeSortsByAmountDesc(t *testing.T) {
	summary := Summarize(testPortfolio(), classify.New(nil))

	for i := 1; i < len(summary.Categories); i++ {
		if summary.Categories[i-1].Amount < summary.Categories[i].Amount {
			t.Fatalf("카테고리가 금액 내림차순이 아님: %+v", summary.Categories)
		}
	}
	for i := 1; i < len(summary.Holdings); i++ {
		if summary.Holdings[i-1].EvalAmount < summary.Holdings[i].EvalAmount {
			t.Fatalf("종목이 평가금액 내림차순이 아님: %+v", summary.Holdings)
		}
	}
}

// 계좌가 없으면 분모가 0이다. NaN 대신 0이 나와야 한다.
func TestSummarizeWithoutAccounts(t *testing.T) {
	summary := Summarize(&Portfolio{Holdings: []domain.Holding{holding("111", "삼성전자", 5000)}}, classify.New(nil))

	for _, detail := range summary.Holdings {
		if detail.Weight != 0 {
			t.Errorf("Weight = %v, want 0", detail.Weight)
		}
	}
}
