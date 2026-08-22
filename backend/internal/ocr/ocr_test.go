package ocr

import (
	"context"
	"errors"
	"testing"
)

func f(v float64) *float64 { return &v }

// RecomputeKRW 는 Gemini 가 이미 evalAmount·profitLoss 를 (잘못된 통화로)
// 채워 놨어도 currentPrice·avgBuyPrice 기준으로 무조건 다시 계산해야 한다.
// 실사용 스크린샷에서 Gemini 가 종목마다 currency 를 빠뜨려 원 단위 계산이
// 안 먹히는 경우가 있었는데, 이건 마스터로 해외 종목임을 확정한 뒤 호출하는
// 별도 경로라 Gemini 의 currency 판단과 무관하게 항상 정확해야 한다.
func TestRecomputeKRWOverridesWrongGeminiValues(t *testing.T) {
	h := ExtractedHolding{
		Quantity:     f(3),
		CurrentPrice: f(347.2327),
		AvgBuyPrice:  f(347.08),
		// Gemini 가 통화를 혼동해 원 단위인 것처럼 잘못 채워 놓은 값
		EvalAmount: f(1041.70),
		ProfitLoss: f(0.45),
		ProfitRate: f(0.04),
	}
	RecomputeKRW(&h, 1446.0)

	wantEval := round2(347.2327 * 1446 * 3)
	if *h.EvalAmount != wantEval {
		t.Errorf("evalAmount = %v, want %v", *h.EvalAmount, wantEval)
	}
	wantBuy := 347.08 * 1446 * 3
	wantProfit := round2(wantEval - wantBuy)
	if *h.ProfitLoss != wantProfit {
		t.Errorf("profitLoss = %v, want %v", *h.ProfitLoss, wantProfit)
	}
	// 실제 주가는 평단가보다 살짝 높은 수준이라 손익률이 크게 나면 안 된다.
	if *h.ProfitRate > 1 || *h.ProfitRate < -1 {
		t.Errorf("profitRate = %v, want 작은 값 (평단가와 현재가가 비슷함)", *h.ProfitRate)
	}
}

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

// 타임아웃을 즉시 실패로 두면 배포 환경에서 다섯 번에 한 번쯤 그대로 실패한다.
// 곧바로 다시 걸면 대개 풀리므로 재시도 대상이어야 한다.
func TestRetryableCoversTimeout(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"타임아웃", context.DeadlineExceeded, true},
		{"요청 전송 중 타임아웃", errors.New(`doRequest: error sending request: Post "https://x": context deadline exceeded`), true},
		{"한도초과", errors.New("Error 429, Message: quota exceeded"), true},
		{"과부하", errors.New("Error 503, Message: This model is currently experiencing high traffic"), true},
		{"잘못된 요청", errors.New("Error 400, Message: Request contains an invalid argument"), false},
		{"지원 종료", errors.New("Error 404, Message: model is no longer available"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := retryable(c.err); got != c.want {
				t.Errorf("retryable(%v) = %v, want %v", c.err, got, c.want)
			}
		})
	}
}
