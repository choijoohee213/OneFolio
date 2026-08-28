package classify

import (
	"testing"

	"github.com/choijoohee213/OneFolio/backend/internal/domain"
	"github.com/choijoohee213/OneFolio/backend/internal/master"
)

// 증권 앱이 ADR 을 그냥 "SK하이닉스" 로 보여주면 이름만으로는 국내 종목과
// 구분되지 않는다. 티커가 함께 잡혔으면 그걸 믿어야 한다.
func TestTickerBeatsAmbiguousName(t *testing.T) {
	listings, err := master.Load()
	if err != nil {
		t.Fatal(err)
	}

	withTicker := New(listings, nil, map[string]string{"SK하이닉스": "SKHY"})
	if got := withTicker.Classify(domain.Holding{Name: "SK하이닉스"}); got != domain.ForeignStock {
		t.Errorf("티커가 SKHY 인데 %q 로 잡혔다", got)
	}

	// 티커가 없으면 이름대로 국내가 맞다
	plain := New(listings, nil, nil)
	if got := plain.Classify(domain.Holding{Name: "SK하이닉스"}); got != domain.DomesticStock {
		t.Errorf("티커가 없으면 국내여야 하는데 %q", got)
	}

	// 국내 종목코드가 잡힌 경우도 그대로 국내
	domestic := New(listings, nil, map[string]string{"SK하이닉스": "000660"})
	if got := domestic.Classify(domain.Holding{Name: "SK하이닉스"}); got != domain.DomesticStock {
		t.Errorf("코드가 000660 인데 %q", got)
	}
}

// 캡처에 적힌 그대로 "SK하이닉스(ADR)" 이 올라오면 티커가 없어도 해외로 가야 한다.
func TestADRNameGoesForeign(t *testing.T) {
	listings, err := master.Load()
	if err != nil {
		t.Fatal(err)
	}
	c := New(listings, nil, nil)
	for _, name := range []string{"SK하이닉스(ADR)", "TSMC(ADR)"} {
		if got := c.Classify(domain.Holding{Name: name}); got != domain.ForeignStock {
			t.Errorf("Classify(%q) = %q, want 개별주(해외)", name, got)
		}
	}
}
