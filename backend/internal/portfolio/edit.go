package portfolio

import "github.com/choijoohee213/OneFolio/backend/internal/domain"

// HoldingEdit 은 잔고파일에서 온 종목의 값을 사용자가 직접 고친 것이다.
// 비워 둔 필드는 파일 값을 그대로 쓴다.
type HoldingEdit struct {
	AccountNumber string   `json:"accountNumber"`
	Name          string   `json:"name"`
	Quantity      *float64 `json:"quantity"`
	EvalAmount    *float64 `json:"evalAmount"`
	AvgBuyPrice   *float64 `json:"avgBuyPrice"`
}

// ApplyEdits 는 수정값을 종목에 덮어쓰고, 덮어쓰기 전 파일 원본을 남긴다.
// 매입금액·손익·손익률·현재가는 받지 않고 여기서 다시 낸다 — 사용자가 수량이나
// 평단만 고쳐도 나머지가 따라 움직여야 표가 앞뒤 안 맞는 숫자를 보여주지 않는다.
func ApplyEdits(p *Portfolio, edits []HoldingEdit) {
	if len(edits) == 0 {
		return
	}

	byKey := make(map[string]HoldingEdit, len(edits))
	for _, edit := range edits {
		byKey[HoldingKey(edit.AccountNumber, edit.Name)] = edit
	}

	accountIdx := make(map[string]int, len(p.Accounts))
	for i, account := range p.Accounts {
		accountIdx[account.Number] = i
	}

	for i, holding := range p.Holdings {
		edit, ok := byKey[HoldingKey(holding.AccountNumber, holding.Name)]
		if !ok {
			continue
		}
		if p.OriginalHoldings == nil {
			p.OriginalHoldings = make(map[string]domain.Holding)
		}
		p.OriginalHoldings[HoldingKey(holding.AccountNumber, holding.Name)] = holding

		before := holding.EvalAmount
		updated := edited(holding, edit)
		p.Holdings[i] = updated

		// 평가금액이 바뀌면 계좌 총액도 같이 옮겨야 비중·현금 계산이 어긋나지 않는다.
		// 계좌 총액은 파일의 "자산총액" 스냅샷이라 종목만 고치면 둘이 따로 논다.
		if idx, ok := accountIdx[holding.AccountNumber]; ok {
			p.Accounts[idx].TotalAsset += updated.EvalAmount - before
		}
	}
}

func edited(holding domain.Holding, edit HoldingEdit) domain.Holding {
	if edit.Quantity != nil {
		holding.Quantity = *edit.Quantity
	}
	if edit.EvalAmount != nil {
		holding.EvalAmount = *edit.EvalAmount
	}
	if edit.AvgBuyPrice != nil {
		holding.AvgBuyPrice = edit.AvgBuyPrice
	}

	// 수량이 0이면 단가를 낼 수 없다. 0으로 나눠 Inf 를 만들지 않는다.
	if holding.Quantity != 0 {
		price := holding.EvalAmount / holding.Quantity
		holding.CurrentPrice = &price
	} else {
		holding.CurrentPrice = nil
	}

	// 평단이 없으면(현금성 등) 취득원가가 없어 손익도 낼 수 없다.
	if holding.AvgBuyPrice == nil {
		holding.BuyAmount, holding.ProfitLoss, holding.ProfitRate = nil, nil, nil
		return holding
	}

	buy := holding.Quantity * *holding.AvgBuyPrice
	profit := holding.EvalAmount - buy
	holding.BuyAmount, holding.ProfitLoss = &buy, &profit
	if buy == 0 {
		holding.ProfitRate = nil
	} else {
		rate := profit / buy * 100
		holding.ProfitRate = &rate
	}
	return holding
}
