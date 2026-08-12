package classify

import (
	"testing"

	"github.com/choijoohee213/OneFolio/backend/internal/domain"
	"github.com/choijoohee213/OneFolio/backend/internal/master"
)

func price(v float64) *float64 { return &v }

func loadListings(t *testing.T) *master.Table {
	t.Helper()
	listings, err := master.Load()
	if err != nil {
		t.Fatalf("종목마스터 로드 실패: %v", err)
	}
	return listings
}

// 실제 보유종목이 마스터 조회만으로 제대로 갈리는지 본다.
func TestClassifyFromMaster(t *testing.T) {
	classifier := New(loadListings(t), nil)

	tests := []struct {
		name string
		want domain.Category
	}{
		{"삼성전자", domain.DomesticStock},
		{"SK하이닉스", domain.DomesticStock},
		{"SK", domain.DomesticStock},

		{"AMD", domain.ForeignStock},
		{"알파벳 A", domain.ForeignStock},
		{"존슨 앤드 존슨", domain.ForeignStock},
		{"코카콜라", domain.ForeignStock},

		{"TIGER 미국S&P500", domain.IndexETF},
		{"TIGER 미국나스닥100", domain.IndexETF},
		{"TIME 미국나스닥100액티브", domain.IndexETF},
		{"SOL 미국배당다우존스", domain.IndexETF},

		{"SOL AI반도체TOP2플러스", domain.ThemeETF},
		{"KODEX 미국AI전력핵심인프라", domain.ThemeETF},
		{"DIREXION SEMICONDUCTOR DAILY 3X", domain.ThemeETF},
		{"PROSHARES QQQ 3X", domain.ThemeETF},
	}

	for _, tt := range tests {
		if got := classifier.Classify(domain.Holding{Name: tt.name}); got != tt.want {
			t.Errorf("Classify(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// 띄어쓰기가 달라도 같은 종목으로 찾아야 한다.
func TestLookupIgnoresSpacing(t *testing.T) {
	classifier := New(loadListings(t), nil)

	for _, name := range []string{"존슨앤드존슨", "알파벳  A"} {
		if got := classifier.Classify(domain.Holding{Name: name}); got != domain.ForeignStock {
			t.Errorf("Classify(%q) = %q, want %q", name, got, domain.ForeignStock)
		}
	}
}

func TestOverrideBeatsMaster(t *testing.T) {
	classifier := New(loadListings(t), map[string]domain.Category{"AMD": domain.DomesticStock})

	if got := classifier.Classify(domain.Holding{Name: "AMD"}); got != domain.DomesticStock {
		t.Errorf("수동 매핑이 마스터를 덮어쓰지 못함: %q", got)
	}
}

// 마스터에 없는 종목은 이름 규칙과 현재가 소수점으로 추정한다.
func TestFallbackForUnlistedName(t *testing.T) {
	classifier := New(master.Empty(), nil)

	tests := []struct {
		name         string
		currentPrice *float64
		want         domain.Category
	}{
		{"신규상장바이오", price(15000), domain.DomesticStock},
		{"신규해외주", price(123456.78), domain.ForeignStock},
		{"KODEX 신규테마", price(10000), domain.ThemeETF},
		{"TIGER 신규나스닥100", price(10000), domain.IndexETF},
		{"이름만레버리지", price(10000), domain.ThemeETF},

		// 브랜드는 접두사로만 인정한다. SOLUS 는 SOL 브랜드가 아니다.
		{"SOLUS첨단소재", price(15000), domain.DomesticStock},
	}

	for _, tt := range tests {
		holding := domain.Holding{Name: tt.name, CurrentPrice: tt.currentPrice}
		if got := classifier.Classify(holding); got != tt.want {
			t.Errorf("Classify(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}
