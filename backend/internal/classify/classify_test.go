package classify

import (
	"testing"

	"github.com/choijoohee213/OneFolio/backend/internal/domain"
)

func price(v float64) *float64 { return &v }

func TestClassifyByRule(t *testing.T) {
	tests := []struct {
		name         string
		currentPrice *float64
		want         domain.Category
	}{
		{name: "삼성전자", currentPrice: price(262500), want: domain.DomesticStock},
		{name: "SK하이닉스", currentPrice: price(1718000), want: domain.DomesticStock},

		// 원화 환산으로 현재가에 소수점이 남으면 해외 상장으로 본다.
		{name: "AMD", currentPrice: price(686179.76), want: domain.ForeignStock},
		{name: "코카콜라", currentPrice: price(126225.94), want: domain.ForeignStock},

		{name: "TIGER 미국S&P500", currentPrice: price(26545), want: domain.IndexETF},
		{name: "TIGER 미국나스닥100", currentPrice: price(179850), want: domain.IndexETF},
		{name: "SOL 미국배당다우존스", currentPrice: price(13885), want: domain.IndexETF},

		{name: "SOL AI반도체TOP2플러스", currentPrice: price(16495), want: domain.ThemeETF},
		{name: "KODEX 미국AI전력핵심인프라", currentPrice: price(20665), want: domain.ThemeETF},
		{name: "KODEX 레버리지", currentPrice: price(20000), want: domain.ThemeETF},

		// 레버리지 판정이 지수 키워드보다 우선한다.
		{name: "PROSHARES QQQ 3X", currentPrice: price(93123.88), want: domain.ThemeETF},
		{name: "DIREXION SEMICONDUCTOR DAILY 3X", currentPrice: price(165322.99), want: domain.ThemeETF},

		// ETF 브랜드는 접두사로만 인정한다. SOLUS 는 SOL 브랜드가 아니다.
		{name: "SOLUS첨단소재", currentPrice: price(15000), want: domain.DomesticStock},
	}

	classifier := New(nil)
	for _, tt := range tests {
		holding := domain.Holding{Name: tt.name, CurrentPrice: tt.currentPrice}
		if got := classifier.Classify(holding); got != tt.want {
			t.Errorf("Classify(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestOverrideBeatsRule(t *testing.T) {
	classifier := New(map[string]domain.Category{"AMD": domain.DomesticStock})

	holding := domain.Holding{Name: "AMD", CurrentPrice: price(686179.76)}
	if got := classifier.Classify(holding); got != domain.DomesticStock {
		t.Errorf("수동 매핑이 규칙을 덮어쓰지 못함: %q", got)
	}
}

// 현재가가 없으면 소수점 판정을 할 수 없다. 국내로 떨어뜨리고 수동 매핑에 맡긴다.
func TestClassifyWithoutPrice(t *testing.T) {
	if got := New(nil).Classify(domain.Holding{Name: "알 수 없는 종목"}); got != domain.DomesticStock {
		t.Errorf("Classify = %q, want %q", got, domain.DomesticStock)
	}
}
