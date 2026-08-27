package portfolio

import (
	"testing"

	"github.com/choijoohee213/OneFolio/backend/internal/classify"
	"github.com/choijoohee213/OneFolio/backend/internal/domain"
	"github.com/choijoohee213/OneFolio/backend/internal/master"
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
	summary := Summarize(testPortfolio(), classify.New(master.Empty(), nil, nil), master.Empty(), nil)

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
	summary := Summarize(testPortfolio(), classify.New(master.Empty(), nil, nil), master.Empty(), nil)

	amount, ok := categoryAmount(summary, domain.Cash)
	if !ok {
		t.Fatal("현금성 카테고리가 없음")
	}
	if amount != 2000 {
		t.Errorf("현금성 = %v, want 2000 (10000 - 8000)", amount)
	}
}

func TestSummarizeWeightsSumTo100(t *testing.T) {
	summary := Summarize(testPortfolio(), classify.New(master.Empty(), nil, nil), master.Empty(), nil)

	var sum float64
	for _, total := range summary.Categories {
		sum += total.Weight
	}
	if diff := sum - 100; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("카테고리 비중 합 = %v, want 100", sum)
	}
}

func TestSummarizeSortsByAmountDesc(t *testing.T) {
	summary := Summarize(testPortfolio(), classify.New(master.Empty(), nil, nil), master.Empty(), nil)

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
	summary := Summarize(&Portfolio{Holdings: []domain.Holding{holding("111", "삼성전자", 5000)}}, classify.New(master.Empty(), nil, nil), master.Empty(), nil)

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
	}, classify.New(master.Empty(), nil, nil), master.Empty(), nil)

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

// 계좌 없이 던져 넣은 종목(ManualHoldingPrefix)은 자기 평가금액만큼 스스로
// 분모를 늘려야 한다. 기존 계좌에서 끌어오면 그 계좌의 현금성이 없던 돈을
// 쓴 것처럼 깎인다.
func TestSummarizeFreestandingHoldingFundsItsOwnWeight(t *testing.T) {
	summary := Summarize(&Portfolio{
		Accounts: []domain.Account{account("111", 8000)},
		Holdings: []domain.Holding{
			priced("111", "삼성전자", 262500, 5000),
			holding(ManualHoldingPrefix+"abc", "예금", 2000),
		},
	}, classify.New(master.Empty(), nil, nil), master.Empty(), nil)

	if summary.CoveredAsset != 10000 {
		t.Fatalf("CoveredAsset = %v, want 10000 (8000 계좌 + 2000 직접 추가)", summary.CoveredAsset)
	}

	cash, ok := categoryAmount(summary, domain.Cash)
	if !ok || cash != 3000 {
		t.Errorf("현금성 = %v(있음=%v), want 3000 (8000 - 5000, 직접 추가분은 건드리지 않음)", cash, ok)
	}

	for _, h := range summary.Holdings {
		if h.Name == "예금" && h.Weight != 20 {
			t.Errorf("예금 비중 = %v, want 20 (2000/10000)", h.Weight)
		}
	}
}

// 계좌 파일 없이 직접 추가한 종목만 있어도 비중이 정상적으로 100% 가 나와야 한다.
func TestSummarizeOnlyFreestandingHoldings(t *testing.T) {
	summary := Summarize(&Portfolio{
		Holdings: []domain.Holding{holding(ManualHoldingPrefix+"a", "예금", 1000), holding(ManualHoldingPrefix+"b", "적금", 3000)},
	}, classify.New(master.Empty(), nil, nil), master.Empty(), nil)

	if summary.CoveredAsset != 4000 {
		t.Fatalf("CoveredAsset = %v, want 4000", summary.CoveredAsset)
	}
	if _, ok := categoryAmount(summary, domain.Cash); ok {
		t.Error("직접 추가한 종목으로 분모가 다 채워지면 현금성이 남으면 안 된다")
	}
}

// 직접 추가한 계좌는 상세 종목이 없어도 항상 집계 대상이고, 종목이 없으면
// 전액이 현금성으로 떨어진다 — 진짜 계좌와 똑같이 계산된다.
func TestSummarizeManualAccountWithoutHoldings(t *testing.T) {
	summary := Summarize(&Portfolio{
		Accounts: []domain.Account{{Number: ManualAccountPrefix + "x", Type: "저축은행", TotalAsset: 8000000}},
	}, classify.New(master.Empty(), nil, nil), master.Empty(), nil)

	if summary.CoveredAsset != 8000000 {
		t.Fatalf("CoveredAsset = %v, want 8000000", summary.CoveredAsset)
	}
	cash, ok := categoryAmount(summary, domain.Cash)
	if !ok || cash != 8000000 {
		t.Errorf("현금성 = %v(있음=%v), want 8000000", cash, ok)
	}
	for _, a := range summary.Accounts {
		if a.Number == ManualAccountPrefix+"x" && !a.Covered {
			t.Error("직접 추가한 계좌는 종목이 없어도 집계 대상이어야 한다")
		}
	}
}

// 사용자가 종목을 직접 "현금성"으로 지정하면, 그 종목 금액은 이미
// amountByCategory[현금성] 에 들어있다. 계좌의 나머지 잔액을 현금성에 더할 때
// 대입(=)으로 덮어쓰면 종목 몫이 사라지고 잔액만 남는다 — 반드시 누적(+=) 이어야 한다.
func TestSummarizeCashCategoryHoldingSurvivesRemainder(t *testing.T) {
	summary := Summarize(&Portfolio{
		Accounts: []domain.Account{{Number: ManualAccountPrefix + "x", Type: "저축은행", TotalAsset: 8000000}},
		Holdings: []domain.Holding{holding(ManualAccountPrefix+"x", "정기예금", 5000000)},
	}, classify.New(master.Empty(), map[string]domain.Category{"정기예금": domain.Cash}, nil), master.Empty(), nil)

	cash, ok := categoryAmount(summary, domain.Cash)
	if !ok || cash != 8000000 {
		t.Fatalf("현금성 = %v(있음=%v), want 8000000 (종목 5,000,000 + 잔액 3,000,000)", cash, ok)
	}
}

// 직접 추가한 계좌에 붙은 종목은 그 계좌 총액에서 자기 몫을 떼어 가고,
// 남는 만큼만 현금성으로 남는다. 계좌 총액과 별개로 분모를 또 늘리면 안 된다.
func TestSummarizeManualAccountWithHolding(t *testing.T) {
	summary := Summarize(&Portfolio{
		Accounts: []domain.Account{{Number: ManualAccountPrefix + "x", Type: "저축은행", TotalAsset: 8000000}},
		Holdings: []domain.Holding{holding(ManualAccountPrefix+"x", "정기예금", 5000000)},
	}, classify.New(master.Empty(), nil, nil), master.Empty(), nil)

	if summary.CoveredAsset != 8000000 {
		t.Fatalf("CoveredAsset = %v, want 8000000 (계좌 총액만, 종목분을 또 더하면 안 됨)", summary.CoveredAsset)
	}
	cash, ok := categoryAmount(summary, domain.Cash)
	if !ok || cash != 3000000 {
		t.Errorf("현금성 = %v(있음=%v), want 3000000 (8000000 - 5000000)", cash, ok)
	}
}

func TestSummarizeMarksCoveredAccounts(t *testing.T) {
	summary := Summarize(&Portfolio{
		Accounts: []domain.Account{account("111", 6000), account("222", 4000)},
		Holdings: []domain.Holding{holding("222", "삼성전자", 1000)},
		Cash:     []domain.Holding{holding("222", "미국달러", -100)},
	}, classify.New(master.Empty(), nil, nil), master.Empty(), nil)

	covered := map[string]bool{}
	for _, account := range summary.Accounts {
		covered[account.Number] = account.Covered
	}
	if covered["222"] != true || covered["111"] != false {
		t.Errorf("Covered 표시가 틀림: %+v", summary.Accounts)
	}
}
