// Package classify 는 보유종목을 자산배분 카테고리로 나눈다.
// 사용자 지정 매핑 > 종목마스터 조회 > 이름 규칙 순으로 우선한다.
package classify

import (
	"math"
	"strings"

	"github.com/choijoohee213/OneFolio/backend/internal/domain"
	"github.com/choijoohee213/OneFolio/backend/internal/master"
)

type Classifier struct {
	listings  *master.Table
	overrides map[string]domain.Category
}

func New(listings *master.Table, overrides map[string]domain.Category) *Classifier {
	return &Classifier{listings: listings, overrides: overrides}
}

func (c *Classifier) Classify(h domain.Holding) domain.Category {
	if category, ok := c.overrides[h.Name]; ok {
		return category
	}
	if listing, ok := c.listings.Lookup(h.Name); ok {
		return FromKind(listing.Kind, h.Name)
	}
	return guess(h)
}

func FromKind(kind master.Kind, name string) domain.Category {
	switch {
	case kind.IsETF():
		return etfStyle(name)
	case kind.IsForeign():
		return domain.ForeignStock
	default:
		return domain.DomesticStock
	}
}

// ETF 라는 것까지는 마스터가 알려주지만 지수추종이냐 레버리지·테마냐는 이름으로 갈라야 한다.
func etfStyle(name string) domain.Category {
	upper := strings.ToUpper(name)
	switch {
	case hasKeyword(upper, leverageKeywords):
		return domain.ThemeETF
	case hasKeyword(upper, indexKeywords):
		return domain.IndexETF
	default:
		return domain.ThemeETF
	}
}

// 마스터에 없는 종목(신규 상장 등)을 위한 폴백.
func guess(h domain.Holding) domain.Category {
	upper := strings.ToUpper(h.Name)
	switch {
	case hasKeyword(upper, leverageKeywords):
		return domain.ThemeETF
	case hasETFBrand(upper):
		return etfStyle(h.Name)
	case hasFractionalPrice(h):
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
func hasETFBrand(upperName string) bool {
	for _, brand := range etfBrands {
		if strings.HasPrefix(upperName, brand+" ") {
			return true
		}
	}
	return false
}

// 해외 종목은 원화로 환산되면서 현재가에 소수점이 남는다(AMD 686,179.76).
func hasFractionalPrice(h domain.Holding) bool {
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
