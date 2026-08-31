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
