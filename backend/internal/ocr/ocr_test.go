package ocr

import "testing"

func f(v float64) *float64 { return &v }

// 평단가·현재가가 달러로 찍힌 종목은 환율로 원화 환산한 뒤에 평가손익을 내야 한다.
// 그렇지 않으면 "70원"과 "$70"을 혼동해 손익률이 터무니없이 커진다.
func TestFillCalculatedFieldsConvertsUSD(t *testing.T) {
	holdings := []ExtractedHolding{{
		Quantity:     f(16),
		CurrentPrice: f(70),
		AvgBuyPrice:  f(65),
		Currency:     "USD",
	}}
	usdKrw := 1400.0
	fillCalculatedFields(holdings, &usdKrw)

	got := holdings[0]
	wantEval := 70.0 * 1400 * 16 // 1,568,000
	if *got.EvalAmount != wantEval {
		t.Errorf("evalAmount = %v, want %v", *got.EvalAmount, wantEval)
	}
	wantProfit := wantEval - 65.0*1400*16 // 80,000
	if *got.ProfitLoss != wantProfit {
		t.Errorf("profitLoss = %v, want %v", *got.ProfitLoss, wantProfit)
	}
}

// 환율을 못 구했으면(usdKrw==nil) 달러 종목은 잘못된 원화 값을 만들지 말고 비워 둔다.
func TestFillCalculatedFieldsSkipsUSDWithoutRate(t *testing.T) {
	holdings := []ExtractedHolding{{
		Quantity:     f(16),
		CurrentPrice: f(70),
		AvgBuyPrice:  f(65),
		Currency:     "USD",
	}}
	fillCalculatedFields(holdings, nil)

	got := holdings[0]
	if got.EvalAmount != nil {
		t.Errorf("evalAmount = %v, want nil (환율 모름)", *got.EvalAmount)
	}
	if got.ProfitLoss != nil {
		t.Errorf("profitLoss = %v, want nil (환율 모름)", *got.ProfitLoss)
	}
}

// 원화 종목은 환율과 무관하게 그대로 계산된다.
func TestFillCalculatedFieldsKRWUnaffected(t *testing.T) {
	holdings := []ExtractedHolding{{
		Quantity:     f(10),
		CurrentPrice: f(75_000),
		AvgBuyPrice:  f(70_000),
		Currency:     "KRW",
	}}
	usdKrw := 1400.0
	fillCalculatedFields(holdings, &usdKrw)

	got := holdings[0]
	if *got.EvalAmount != 750_000 {
		t.Errorf("evalAmount = %v, want 750000", *got.EvalAmount)
	}
	if *got.ProfitLoss != 50_000 {
		t.Errorf("profitLoss = %v, want 50000", *got.ProfitLoss)
	}
}

// evalAmount·profitLoss 를 Gemini 가 이미 원화로 채워 줬으면(달러 종목이라도)
// 그 값을 그대로 쓴다 — 환율을 몰라도 계산이 막히면 안 된다.
func TestFillCalculatedFieldsUsesGivenKRWValuesEvenForUSD(t *testing.T) {
	holdings := []ExtractedHolding{{
		Quantity:     f(16),
		CurrentPrice: f(70),
		AvgBuyPrice:  f(65),
		Currency:     "USD",
		EvalAmount:   f(1_568_000),
		ProfitLoss:   f(80_000),
	}}
	fillCalculatedFields(holdings, nil)

	got := holdings[0]
	if *got.EvalAmount != 1_568_000 {
		t.Errorf("evalAmount = %v, want 1568000 (그대로 유지)", *got.EvalAmount)
	}
	if *got.ProfitLoss != 80_000 {
		t.Errorf("profitLoss = %v, want 80000 (그대로 유지)", *got.ProfitLoss)
	}
}
