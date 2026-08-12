package portfolio

import (
	"testing"

	"github.com/choijoohee213/OneFolio/backend/internal/domain"
)

func amount(v float64) *float64 { return &v }

func fileHolding() domain.Holding {
	return domain.Holding{
		AccountNumber: "111",
		Name:          "삼성전자",
		Quantity:      10,
		CurrentPrice:  amount(75_000),
		AvgBuyPrice:   amount(70_000),
		BuyAmount:     amount(700_000),
		EvalAmount:    750_000,
		ProfitLoss:    amount(50_000),
		ProfitRate:    amount(7.142857142857143),
	}
}

// 평가금액만 고쳐도 손익·손익률·현재가가 따라 움직여야 한다. 안 그러면 표에
// 앞뒤 안 맞는 숫자가 남는다.
func TestApplyEditsRecalculates(t *testing.T) {
	p := &Portfolio{Holdings: []domain.Holding{fileHolding()}}
	ApplyEdits(p, []HoldingEdit{{AccountNumber: "111", Name: "삼성전자", EvalAmount: amount(900_000)}})

	got := p.Holdings[0]
	if got.EvalAmount != 900_000 {
		t.Errorf("평가금액 = %v, want 900000", got.EvalAmount)
	}
	if *got.CurrentPrice != 90_000 {
		t.Errorf("현재가 = %v, want 90000 (900000/10)", *got.CurrentPrice)
	}
	if *got.ProfitLoss != 200_000 {
		t.Errorf("평가손익 = %v, want 200000 (900000-700000)", *got.ProfitLoss)
	}
	// 부동소수점이라 정확히 같기를 기대하지 않는다.
	if want := 28.5714; *got.ProfitRate-want > 0.001 || want-*got.ProfitRate > 0.001 {
		t.Errorf("손익률 = %v, want 약 %v", *got.ProfitRate, want)
	}
}

// 수량을 고치면 매입금액도 평단 기준으로 다시 나야 한다.
func TestApplyEditsRecalculatesBuyAmountFromQuantity(t *testing.T) {
	p := &Portfolio{Holdings: []domain.Holding{fileHolding()}}
	ApplyEdits(p, []HoldingEdit{{AccountNumber: "111", Name: "삼성전자", Quantity: amount(20)}})

	got := p.Holdings[0]
	if *got.BuyAmount != 1_400_000 {
		t.Errorf("매입금액 = %v, want 1400000 (20 × 70000)", *got.BuyAmount)
	}
	// 평가금액은 안 고쳤으니 그대로고, 수량이 늘었으니 현재가는 내려간다.
	if *got.CurrentPrice != 37_500 {
		t.Errorf("현재가 = %v, want 37500 (750000/20)", *got.CurrentPrice)
	}
}

// 고치기 전 파일 원본을 남겨야 화면이 "파일이 그새 바뀌었는지" 판별할 수 있다.
func TestApplyEditsKeepsOriginal(t *testing.T) {
	p := &Portfolio{Holdings: []domain.Holding{fileHolding()}}
	ApplyEdits(p, []HoldingEdit{{AccountNumber: "111", Name: "삼성전자", EvalAmount: amount(900_000)}})

	original, ok := p.OriginalHoldings[HoldingKey("111", "삼성전자")]
	if !ok {
		t.Fatal("원본이 남지 않았다")
	}
	if original.EvalAmount != 750_000 {
		t.Errorf("원본 평가금액 = %v, want 750000", original.EvalAmount)
	}
}

// 계좌가 다르면 같은 종목명이라도 건드리면 안 된다.
func TestApplyEditsMatchesAccountToo(t *testing.T) {
	other := fileHolding()
	other.AccountNumber = "222"
	p := &Portfolio{Holdings: []domain.Holding{fileHolding(), other}}

	ApplyEdits(p, []HoldingEdit{{AccountNumber: "111", Name: "삼성전자", EvalAmount: amount(900_000)}})

	if p.Holdings[1].EvalAmount != 750_000 {
		t.Errorf("다른 계좌 종목이 바뀜: %v", p.Holdings[1].EvalAmount)
	}
}

// 평단이 없는 종목(현금성 등)은 취득원가가 없으니 손익도 낼 수 없다.
func TestApplyEditsWithoutAvgBuyPrice(t *testing.T) {
	cash := domain.Holding{AccountNumber: "111", Name: "미국달러", Quantity: 100, EvalAmount: 140_000}
	p := &Portfolio{Holdings: []domain.Holding{cash}}

	ApplyEdits(p, []HoldingEdit{{AccountNumber: "111", Name: "미국달러", EvalAmount: amount(150_000)}})

	got := p.Holdings[0]
	if got.EvalAmount != 150_000 {
		t.Errorf("평가금액 = %v, want 150000", got.EvalAmount)
	}
	if got.BuyAmount != nil || got.ProfitLoss != nil || got.ProfitRate != nil {
		t.Errorf("평단이 없으면 손익을 만들면 안 된다: %+v", got)
	}
}

// 수량 0 으로 고쳐도 0 으로 나눠 Inf 를 만들면 안 된다.
func TestApplyEditsWithZeroQuantity(t *testing.T) {
	p := &Portfolio{Holdings: []domain.Holding{fileHolding()}}
	ApplyEdits(p, []HoldingEdit{{AccountNumber: "111", Name: "삼성전자", Quantity: amount(0)}})

	if got := p.Holdings[0]; got.CurrentPrice != nil {
		t.Errorf("현재가 = %v, want nil (수량 0)", *got.CurrentPrice)
	}
}
