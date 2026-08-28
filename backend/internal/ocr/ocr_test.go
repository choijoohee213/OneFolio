package ocr

import (
	"context"
	"errors"
	"testing"
)

func f(v float64) *float64 { return &v }

// 화면에서 읽은 값은 그대로 둔다. 예전에는 통화를 혼동했을까 봐 환율로 다시
// 계산해 덮어썼는데, 그 탓에 화면에 원화로 적힌 금액이 우리 환율로 바뀌었다.
func TestFillCalculatedKeepsValuesFromScreen(t *testing.T) {
	holdings := []ExtractedHolding{{
		Currency:     "USD",
		Quantity:     f(0.5237),
		CurrentPrice: f(182.44),
		// 화면에 원화로 적혀 있던 값
		EvalAmount: f(133180),
		ProfitLoss: f(12400),
	}}
	FillCalculated(holdings, f(1446))

	h := holdings[0]
	if *h.EvalAmount != 133180 {
		t.Errorf("화면 값이 바뀌었다: evalAmount = %v, want 133180", *h.EvalAmount)
	}
	if *h.ProfitLoss != 12400 {
		t.Errorf("화면 값이 바뀌었다: profitLoss = %v, want 12400", *h.ProfitLoss)
	}
}

// 화면에 없던 값은 계산해서 채운다.
func TestFillCalculatedFillsMissing(t *testing.T) {
	holdings := []ExtractedHolding{{
		Currency:     "USD",
		Quantity:     f(2),
		CurrentPrice: f(100),
		AvgBuyPrice:  f(80),
	}}
	FillCalculated(holdings, f(1400))

	h := holdings[0]
	if h.EvalAmount == nil || *h.EvalAmount != 100*1400*2 {
		t.Errorf("evalAmount 가 안 채워졌다: %v", h.EvalAmount)
	}
	if h.ProfitLoss == nil || *h.ProfitLoss != (100*1400*2)-(80*1400*2) {
		t.Errorf("profitLoss 가 안 채워졌다: %v", h.ProfitLoss)
	}
}

func TestFillCalculatedConvertsUSD(t *testing.T) {
	holdings := []ExtractedHolding{{
		Quantity:     f(16),
		CurrentPrice: f(70),
		AvgBuyPrice:  f(65),
		Currency:     "USD",
	}}
	usdKrw := 1400.0
	FillCalculated(holdings, &usdKrw)

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
func TestFillCalculatedSkipsUSDWithoutRate(t *testing.T) {
	holdings := []ExtractedHolding{{
		Quantity:     f(16),
		CurrentPrice: f(70),
		AvgBuyPrice:  f(65),
		Currency:     "USD",
	}}
	FillCalculated(holdings, nil)

	got := holdings[0]
	if got.EvalAmount != nil {
		t.Errorf("evalAmount = %v, want nil (환율 모름)", *got.EvalAmount)
	}
	if got.ProfitLoss != nil {
		t.Errorf("profitLoss = %v, want nil (환율 모름)", *got.ProfitLoss)
	}
}

// 원화 종목은 환율과 무관하게 그대로 계산된다.
func TestFillCalculatedKRWUnaffected(t *testing.T) {
	holdings := []ExtractedHolding{{
		Quantity:     f(10),
		CurrentPrice: f(75_000),
		AvgBuyPrice:  f(70_000),
		Currency:     "KRW",
	}}
	usdKrw := 1400.0
	FillCalculated(holdings, &usdKrw)

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
func TestFillCalculatedUsesGivenKRWValuesEvenForUSD(t *testing.T) {
	holdings := []ExtractedHolding{{
		Quantity:     f(16),
		CurrentPrice: f(70),
		AvgBuyPrice:  f(65),
		Currency:     "USD",
		EvalAmount:   f(1_568_000),
		ProfitLoss:   f(80_000),
	}}
	FillCalculated(holdings, nil)

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
