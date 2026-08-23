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

// ETF 라는 것까지는 마스터가 알려주지만 무엇을 담았는지는 이름으로 갈라야 한다.
//
// 레버리지를 가장 먼저 본다. 국고채10년레버리지처럼 기초자산이 채권이어도
// 배율이 붙으면 위험 성격은 레버리지 쪽이라 그렇게 봐야 한다.
func etfStyle(name string) domain.Category {
	upper := strings.ToUpper(name)
	switch {
	case hasKeyword(upper, leverageKeywords):
		return domain.Leverage
	case hasKeyword(upper, bondKeywords):
		return domain.Bond
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
		return domain.Leverage
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
	leverageKeywords = []string{"레버리지", "인버스", "곱버스", "2X", "3X", "2배", "3배", "-1X", "-2X", "-3X"}
	// 국고채는 국채를 품고 있지 않아 따로 적어야 한다. 금융채·은행채처럼
	// 발행 주체가 앞에 붙는 것들도 마찬가지다.
	bondKeywords = []string{
		"국채", "국고채", "국공채", "회사채", "금융채", "은행채", "특수채", "통안채",
		"산금채", "여전채", "물가채", "전단채", "채권", "크레딧",
		"CD금리", "KOFR", "SOFR", "단기자금", "머니마켓", "MMF",
	}
	indexKeywords = []string{"S&P500", "S&P 500", "나스닥100", "코스피200", "KOSPI200", "다우존스", "MSCI", "TOTAL"}
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
