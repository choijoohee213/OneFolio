package ocr

import "testing"

func TestResolveCurrency(t *testing.T) {
	tests := []struct {
		name     string
		holding  ExtractedHolding
		domestic bool
		want     string
	}{
		{
			// 해외주식 화면이 원가는 달러로, 손익은 원화로 보여 주는 경우.
			// 주당 평가금액을 현재가로 나누면 환율 규모가 나온다.
			name: "원가는 달러 손익은 원화",
			holding: ExtractedHolding{
				Quantity: f(1), CurrentPrice: f(500), AvgBuyPrice: f(540),
				ProfitLoss: f(-56000), ProfitRate: f(-7.4),
			},
			want: usd,
		},
		{
			// 같은 해외 종목을 원화 보기로 둔 화면. 나눈 값이 1 근처다.
			name: "해외 종목이지만 화면이 원화",
			holding: ExtractedHolding{
				Quantity: f(1), CurrentPrice: f(700000), AvgBuyPrice: f(760000),
				ProfitLoss: f(-60000), ProfitRate: f(-7.9),
			},
			want: krw,
		},
		{
			// 모델이 달러라고 잘못 읽어도 캡처 안의 값이 이긴다.
			name: "모델 판단이 값과 어긋나면 값을 믿는다",
			holding: ExtractedHolding{
				Currency: usd,
				Quantity: f(2), CurrentPrice: f(300000), EvalAmount: f(600000),
			},
			want: krw,
		},
		{
			name:     "국내 종목은 무조건 원화",
			holding:  ExtractedHolding{Currency: usd, Quantity: f(1), CurrentPrice: f(500), EvalAmount: f(700000)},
			domestic: true,
			want:     krw,
		},
		{
			name:    "비교할 값이 없으면 모델 판단을 쓴다",
			holding: ExtractedHolding{Currency: usd, CurrentPrice: f(500)},
			want:    usd,
		},
		{
			name:    "비교할 값도 통화 표시도 없으면 원화",
			holding: ExtractedHolding{CurrentPrice: f(500)},
			want:    krw,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveCurrency(tt.holding, tt.domestic); got != tt.want {
				t.Errorf("ResolveCurrency() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestResolveCurrenciesSharesScreenCurrency(t *testing.T) {
	// 손익이 정확히 0 이면 손익률도 0 이라 나눠 볼 수가 없다. 같은 화면의 다른
	// 줄이 달러로 가려졌으면 이 줄도 달러다 — 아니면 값이 환율배만큼 어긋난다.
	holdings := []ExtractedHolding{
		{Name: "본전", Quantity: f(4), CurrentPrice: f(87.74), AvgBuyPrice: f(87.74),
			ProfitLoss: f(0), ProfitRate: f(0)},
		{Name: "손실", Quantity: f(2), CurrentPrice: f(258), AvgBuyPrice: f(265.15),
			ProfitLoss: f(-20290), ProfitRate: f(-2.70)},
	}
	ResolveCurrencies(holdings, []bool{false, false})

	if holdings[1].Currency != usd {
		t.Fatalf("나눠 볼 수 있는 줄이 %s 로 잡혔다", holdings[1].Currency)
	}
	if holdings[0].Currency != usd {
		t.Errorf("같은 화면인데 %s 로 잡혔다 — 달러여야 한다", holdings[0].Currency)
	}
}

func TestResolveCurrenciesKeepsDomesticInKrw(t *testing.T) {
	// 국내 종목은 화면에 달러 종목이 섞여 있어도 원화다.
	holdings := []ExtractedHolding{
		{Name: "국내", Quantity: f(9), CurrentPrice: f(274500), ProfitLoss: f(0), ProfitRate: f(0)},
		{Name: "해외", Quantity: f(2), CurrentPrice: f(258), ProfitLoss: f(-20290), ProfitRate: f(-2.70)},
	}
	ResolveCurrencies(holdings, []bool{true, false})

	if holdings[0].Currency != krw {
		t.Errorf("국내 종목이 %s 로 잡혔다", holdings[0].Currency)
	}
	if holdings[1].Currency != usd {
		t.Errorf("해외 종목이 %s 로 잡혔다", holdings[1].Currency)
	}
}

func TestFillCalculatedPrefersScreenPriceForKrw(t *testing.T) {
	// 원화 화면은 현재가 × 수량이 평가금액과 딱 맞는다. 손익률(소수점 둘째
	// 자리)에서 역산하면 몇 천 원 어긋난다.
	holdings := []ExtractedHolding{{
		Currency: krw, Quantity: f(9), CurrentPrice: f(274500),
		ProfitLoss: f(29500), ProfitRate: f(1.21),
	}}
	FillCalculated(holdings, nil)

	const want = 274500.0 * 9
	if holdings[0].EvalAmount == nil || *holdings[0].EvalAmount != want {
		t.Errorf("평가금액 = %v, want %v", holdings[0].EvalAmount, want)
	}
}

func TestFillCalculatedUsesBrokerFxForUsd(t *testing.T) {
	// 달러 화면은 평가손익·손익률이 이미 원화라, 역산하면 증권사 환율이 실린다.
	// 우리 환율로 곱하면 그만큼 어긋난다.
	holdings := []ExtractedHolding{{
		Currency: usd, Quantity: f(10), CurrentPrice: f(200),
		ProfitLoss: f(280000), ProfitRate: f(11.11),
	}}
	rate := 1300.0
	FillCalculated(holdings, &rate)

	got := *holdings[0].EvalAmount
	if impliedFx := got / 10 / 200; impliedFx < 1390 || impliedFx > 1410 {
		t.Errorf("평가금액 %v 가 함축하는 환율 %.1f — 증권사 환율(약 1400)이어야 한다", got, impliedFx)
	}
}
