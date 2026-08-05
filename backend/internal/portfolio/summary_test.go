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
	summary := Summarize(testPortfolio(), classify.New(nil, nil))

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
	summary := Summarize(testPortfolio(), classify.New(nil, nil))

	amount, ok := categoryAmount(summary, domain.Cash)
	if !ok {
		t.Fatal("현금성 카테고리가 없음")
	}
	if amount != 2000 {
		t.Errorf("현금성 = %v, want 2000 (10000 - 8000)", amount)
	}
}

func TestSummarizeWeightsSumTo100(t *testing.T) {
	summary := Summarize(testPortfolio(), classify.New(nil, nil))

	var sum float64
	for _, total := range summary.Categories {
		sum += total.Weight
	}
	if diff := sum - 100; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("카테고리 비중 합 = %v, want 100", sum)
	}
}

func TestSummarizeSortsByAmountDesc(t *testing.T) {
	summary := Summarize(testPortfolio(), classify.New(nil, nil))

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
	summary := Summarize(&Portfolio{Holdings: []domain.Holding{holding("111", "삼성전자", 5000)}}, classify.New(nil, nil))

	for _, detail := range summary.Holdings {
		if detail.Weight != 0 {
			t.Errorf("Weight = %v, want 0", detail.Weight)
		}
	}
	if _, ok := categoryAmount(summary, domain.Cash); ok {
		t.Error("계좌 총액을 모르면 현금성을 만들 수 없다")
	}
}

// 계좌 하나만 올려도 파일에는 세 계좌 요약이 다 들어있다. 올리지 않은 계좌의
// 자산이 현금성으로 새면 안 되고, 비중 분모도 올린 계좌로 좁혀야 한다.
func TestSummarizePartialUpload(t *testing.T) {
	summary := Summarize(&Portfolio{
		Accounts: []domain.Account{account("111", 6000), account("222", 4000), account("333", 3000)},
		Holdings: []domain.Holding{priced("333", "TIGER 미국S&P500", 26545, 2800)},
	}, classify.New(nil, nil))

	if summary.TotalAsset != 13000 {
		t.Errorf("TotalAsset = %v, want 13000 (파일의 전체 계좌)", summary.TotalAsset)
	}
	if summary.CoveredAsset != 3000 {
		t.Errorf("CoveredAsset = %v, want 3000 (종목이 올라온 계좌만)", summary.CoveredAsset)
	}

	cash, ok := categoryAmount(summary, domain.Cash)
	if !ok {
		t.Fatal("현금성 카테고리가 없음")
	}
	if cash != 200 {
		t.Errorf("현금성 = %v, want 200 (3000 - 2800). 올리지 않은 계좌가 섞이면 안 된다", cash)
	}

	if want := 2800.0 / 3000 * 100; summary.Holdings[0].Weight != want {
		t.Errorf("종목 비중 = %v, want %v", summary.Holdings[0].Weight, want)
	}
}

func TestSummarizeMarksCoveredAccounts(t *testing.T) {
	summary := Summarize(&Portfolio{
		Accounts: []domain.Account{account("111", 6000), account("222", 4000)},
		Holdings: []domain.Holding{holding("222", "삼성전자", 1000)},
		Cash:     []domain.Holding{holding("222", "미국달러", -100)},
	}, classify.New(nil, nil))

	covered := map[string]bool{}
	for _, account := range summary.Accounts {
		covered[account.Number] = account.Covered
	}
	if covered["222"] != true || covered["111"] != false {
		t.Errorf("Covered 표시가 틀림: %+v", summary.Accounts)
	}
}
