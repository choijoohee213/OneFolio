package portfolio

import (
	"sort"

	"github.com/choijoohee213/OneFolio/backend/internal/classify"
	"github.com/choijoohee213/OneFolio/backend/internal/domain"
)

type Summary struct {
	TotalAsset float64         `json:"totalAsset"`
	Categories []CategoryTotal `json:"categories"`
	Holdings   []HoldingDetail `json:"holdings"`
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

// Summarize 는 계좌 자산총액 합을 분모로 카테고리별·종목별 비중을 낸다.
// 종목으로 잡히지 않는 잔액(예수금, 외화 등)은 전부 현금성으로 본다.
func Summarize(p *Portfolio, classifier *classify.Classifier) Summary {
	summary := Summary{TotalAsset: totalAsset(p.Accounts)}

	amountByCategory := make(map[domain.Category]float64)
	var holdingsTotal float64

	for _, holding := range p.Holdings {
		category := classifier.Classify(holding)
		amountByCategory[category] += holding.EvalAmount
		holdingsTotal += holding.EvalAmount

		summary.Holdings = append(summary.Holdings, HoldingDetail{
			Holding:  holding,
			Category: category,
			Weight:   ratio(holding.EvalAmount, summary.TotalAsset),
		})
	}

	if cash := summary.TotalAsset - holdingsTotal; cash != 0 {
		amountByCategory[domain.Cash] = cash
	}

	for category, amount := range amountByCategory {
		summary.Categories = append(summary.Categories, CategoryTotal{
			Category: category,
			Amount:   amount,
			Weight:   ratio(amount, summary.TotalAsset),
		})
	}

	sortByAmountDesc(summary.Categories, summary.Holdings)
	return summary
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
