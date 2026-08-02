// Package classify 는 보유종목을 자산배분 카테고리로 나눈다.
// 자동 규칙으로 1차 분류하고, 사용자가 지정한 매핑이 있으면 그쪽이 우선한다.
package classify

import (
	"math"
	"strings"

	"github.com/choijoohee213/OneFolio/backend/internal/domain"
)

type Classifier struct {
	overrides map[string]domain.Category
}

func New(overrides map[string]domain.Category) *Classifier {
	return &Classifier{overrides: overrides}
}

func (c *Classifier) Classify(h domain.Holding) domain.Category {
	if category, ok := c.overrides[h.Name]; ok {
		return category
	}

	name := strings.ToUpper(h.Name)
	switch {
	case hasKeyword(name, leverageKeywords):
		return domain.ThemeETF
	case isETF(name):
		if hasKeyword(name, indexKeywords) {
			return domain.IndexETF
		}
		return domain.ThemeETF
	case isForeignListed(h):
		return domain.ForeignStock
	default:
		return domain.DomesticStock
	}
}

var (
	etfBrands = []string{
		"KODEX", "TIGER", "SOL", "ACE", "PLUS", "RISE", "TIME", "KOSEF",
		"HANARO", "KBSTAR", "ARIRANG", "TIMEFOLIO",
		"DIREXION", "PROSHARES", "ISHARES", "VANGUARD", "SPDR", "INVESCO",
	}
	leverageKeywords = []string{"레버리지", "인버스", "곱버스", "2X", "3X", "2배", "3배"}
	indexKeywords    = []string{"S&P500", "S&P 500", "나스닥100", "코스피200", "KOSPI200", "다우존스", "MSCI", "TOTAL"}
)

// ETF 이름은 "브랜드 + 상품명" 꼴이라 접두사로 본다. 부분일치로 하면
// SOLUS첨단소재 같은 개별주가 SOL 브랜드로 잡힌다.
func isETF(upperName string) bool {
	for _, brand := range etfBrands {
		if strings.HasPrefix(upperName, brand+" ") {
			return true
		}
	}
	return false
}

// 해외 종목은 원화로 환산되면서 현재가에 소수점이 남는다(AMD 686,179.76).
// 국내 상장 종목의 현재가는 항상 정수다. 틀리면 수동 매핑으로 덮어쓴다.
func isForeignListed(h domain.Holding) bool {
	return h.CurrentPrice != nil && *h.CurrentPrice != math.Trunc(*h.CurrentPrice)
}

func hasKeyword(upperName string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(upperName, keyword) {
			return true
		}
	}
	return false
}
