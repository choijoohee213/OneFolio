package portfolio

import (
	"sort"

	"github.com/choijoohee213/OneFolio/backend/internal/classify"
	"github.com/choijoohee213/OneFolio/backend/internal/domain"
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
}

// Summarize 는 종목 상세가 올라온 계좌들의 자산총액 합을 분모로 비중을 낸다.
// 그 계좌들에서 종목으로 잡히지 않는 잔액(예수금, 외화 등)은 현금성으로 본다.
func Summarize(p *Portfolio, classifier *classify.Classifier) Summary {
	covered := coveredAccounts(p)

	summary := Summary{
		TotalAsset: totalAsset(p.Accounts),
		Accounts:   accountSummaries(p.Accounts, covered),
	}
	for _, account := range summary.Accounts {
		if account.Covered {
			summary.CoveredAsset += account.TotalAsset
		}
	}

	amountByCategory := make(map[domain.Category]float64)
	var holdingsTotal float64

	for _, holding := range p.Holdings {
		category := classifier.Classify(holding)
		amountByCategory[category] += holding.EvalAmount
		holdingsTotal += holding.EvalAmount

		summary.Holdings = append(summary.Holdings, HoldingDetail{
			Holding:  holding,
			Category: category,
			Weight:   ratio(holding.EvalAmount, summary.CoveredAsset),
		})
	}

	// 계좌 총액을 모르면 잔액을 뺄 기준이 없다.
	if cash := summary.CoveredAsset - holdingsTotal; summary.CoveredAsset > 0 && cash != 0 {
		amountByCategory[domain.Cash] = cash
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
			Covered:    covered[account.Number],
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
