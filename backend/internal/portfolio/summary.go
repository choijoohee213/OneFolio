package portfolio

import (
	"sort"

	"github.com/choijoohee213/OneFolio/backend/internal/classify"
	"github.com/choijoohee213/OneFolio/backend/internal/domain"
	"github.com/choijoohee213/OneFolio/backend/internal/master"
)

type Summary struct {
	// CoveredAsset 는 종목 상세까지 올라온 계좌들의 자산총액 합이고, 모든 비중의 분모다.
	// TotalAsset 은 파일에 적힌 전체 계좌 합이라 올리지 않은 계좌까지 들어있다.
	// 둘을 구분하지 않으면 안 올린 계좌의 자산이 현금성으로 잘못 잡힌다.
	CoveredAsset float64          `json:"coveredAsset"`
	TotalAsset   float64          `json:"totalAsset"`
	Accounts     []AccountSummary `json:"accounts"`
	Categories   []CategoryTotal  `json:"categories"`
	Holdings     []HoldingDetail  `json:"holdings"`

	// Sources 는 업로드한 파일이 각각 어느 계좌를 담당했는지 알려준다.
	// 요청에 보낸 파일 순서와 같다.
	Sources []Source `json:"sources"`

	Unmatched []string `json:"unmatched,omitempty"`
}

type Source struct {
	FileName       string   `json:"fileName"`
	AccountNumbers []string `json:"accountNumbers"`
}

type AccountSummary struct {
	Number     string  `json:"number"`
	Type       string  `json:"type"`
	TotalAsset float64 `json:"totalAsset"`
	Covered    bool    `json:"covered"`
}

type CategoryTotal struct {
	Category domain.Category `json:"category"`
	Amount   float64         `json:"amount"`
	Weight   float64         `json:"weight"`
}

type HoldingDetail struct {
	domain.Holding
	Category domain.Category `json:"category"`
	Weight   float64         `json:"weight"`
	Code     string          `json:"code,omitempty"`
	Market   master.Market   `json:"market,omitempty"`

	// Original 은 사용자가 값을 고친 종목의 잔고파일 원본이다. 고치지 않았으면 비어 있다.
	// 화면이 "내가 고칠 때 보던 파일 값"과 비교해 파일이 그새 바뀌었는지 가린다.
	Original *domain.Holding `json:"original,omitempty"`
}

// Summarize 는 종목 상세가 올라온 계좌들의 자산총액 합을 분모로 비중을 낸다.
// 그 계좌들에서 종목으로 잡히지 않는 잔액(예수금, 외화 등)은 현금성으로 본다.
func Summarize(p *Portfolio, classifier *classify.Classifier, listings *master.Table, stockMappings map[string]string) Summary {
	covered := coveredAccounts(p)

	summary := Summary{
		TotalAsset: totalAsset(p.Accounts),
		Accounts:   accountSummaries(p.Accounts, covered),
		// 계좌만 추가되고 종목이 하나도 없으면 아래 루프가 한 번도 안 돈다.
		// nil 슬라이스로 두면 JSON 에서 null 로 나가 프런트의 배열 처리가 깨진다.
		Holdings:   []HoldingDetail{},
		Categories: []CategoryTotal{},
	}
	for _, account := range summary.Accounts {
		if account.Covered {
			summary.CoveredAsset += account.TotalAsset
		}
	}
	// 계좌 없이 던져 넣은 종목은 자기 평가금액만큼 스스로 분모를 늘린다.
	// 기존 계좌 총액에서 끌어오면 그 계좌의 현금성이 없던 돈을 쓴 것처럼 깎인다.
	// 직접 추가한 계좌에 붙은 종목은 이미 그 계좌의 TotalAsset 이 분모를
	// 채우고 있으니 여기서 또 더하지 않는다.
	for _, holding := range p.Holdings {
		if IsManualHolding(holding.AccountNumber) {
			summary.CoveredAsset += holding.EvalAmount
		}
	}

	amountByCategory := make(map[domain.Category]float64)
	var holdingsTotal float64

	unmatchedSet := make(map[string]bool)

	for _, holding := range p.Holdings {
		category := classifier.Classify(holding)
		amountByCategory[category] += holding.EvalAmount
		holdingsTotal += holding.EvalAmount

		detail := HoldingDetail{
			Holding:  holding,
			Category: category,
			Weight:   ratio(holding.EvalAmount, summary.CoveredAsset),
		}

		// 종목코드를 먼저 본다 — 이름은 국내와 해외에 같은 것이 있을 수 있어
		// (SK하이닉스와 그 ADR) 코드가 더 확실한 신원이다.
		matched := false
		if code, has := stockMappings[holding.Name]; has && code != "" {
			if entry, ok := listings.LookupByCode(code); ok {
				detail.Code = entry.Code
				detail.Market = entry.Market
				matched = true
			}
		}
		if listing, ok := listings.Lookup(holding.Name); ok && !matched {
			detail.Code = listing.Code
			detail.Market = listing.Market
			matched = true
		}
		if !matched && !IsManualHolding(holding.AccountNumber) {
			unmatchedSet[holding.Name] = true
		}

		if original, ok := p.OriginalHoldings[HoldingKey(holding.AccountNumber, holding.Name)]; ok {
			detail.Original = &original
		}
		summary.Holdings = append(summary.Holdings, detail)
	}

	for name := range unmatchedSet {
		summary.Unmatched = append(summary.Unmatched, name)
	}
	sort.Strings(summary.Unmatched)

	// 계좌 총액을 모르면 잔액을 뺄 기준이 없다. += 여야 한다 — 사용자가 종목을
	// 직접 "현금성"으로 지정하면 그 종목 몫이 이미 amountByCategory[Cash] 에
	// 들어있는데, = 로 덮어쓰면 잔액만 남고 그 종목 금액이 사라진다.
	if cash := summary.CoveredAsset - holdingsTotal; summary.CoveredAsset > 0 && cash != 0 {
		amountByCategory[domain.Cash] += cash
	}

	for category, amount := range amountByCategory {
		summary.Categories = append(summary.Categories, CategoryTotal{
			Category: category,
			Amount:   amount,
			Weight:   ratio(amount, summary.CoveredAsset),
		})
	}

	sortByAmountDesc(summary.Categories, summary.Holdings)
	return summary
}

// 종목이나 현금성 행이 하나라도 올라온 계좌만 집계 대상이다. 잔고파일은 어느 것을
// 받아도 전체 계좌현황을 담고 있어서, 계좌 목록만으로는 무엇을 올렸는지 알 수 없다.
func coveredAccounts(p *Portfolio) map[string]bool {
	covered := make(map[string]bool)
	for _, holding := range p.Holdings {
		covered[holding.AccountNumber] = true
	}
	for _, cash := range p.Cash {
		covered[cash.AccountNumber] = true
	}
	return covered
}

func accountSummaries(accounts []domain.Account, covered map[string]bool) []AccountSummary {
	summaries := make([]AccountSummary, 0, len(accounts))
	for _, account := range accounts {
		summaries = append(summaries, AccountSummary{
			Number:     account.Number,
			Type:       account.Type,
			TotalAsset: account.TotalAsset,
			// 직접 추가한 계좌는 상세 종목이 없어도 항상 집계 대상이다. 사용자가
			// 총액을 직접 써넣은 것이라 "파일을 더 올려야 아는" 상태가 아니다.
			Covered: covered[account.Number] || IsManualAccount(account.Number),
		})
	}
	return summaries
}

func totalAsset(accounts []domain.Account) float64 {
	var total float64
	for _, a := range accounts {
		total += a.TotalAsset
	}
	return total
}

func ratio(amount, total float64) float64 {
	if total == 0 {
		return 0
	}
	return amount / total * 100
}

func sortByAmountDesc(categories []CategoryTotal, holdings []HoldingDetail) {
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Amount > categories[j].Amount
	})
	sort.Slice(holdings, func(i, j int) bool {
		return holdings[i].EvalAmount > holdings[j].EvalAmount
	})
}
