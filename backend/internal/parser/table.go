package parser

import (
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/choijoohee213/OneFolio/backend/internal/domain"
)

type tableKind int

const (
	unknownTable tableKind = iota
	accountsTable
	holdingsTable
)

// 전체 계좌현황 컬럼
const (
	accNumber = iota
	accType
	accAlias
	accTotalAsset
	accWithdrawable
	accShortcut
	accColumns
)

// 상품보유현황 컬럼
const (
	hdName = iota
	hdQuantity
	hdCurrentPrice
	hdAvgBuyPrice
	hdBuyAmount
	hdEvalAmount
	hdProfitLoss
	hdProfitRate
	hdColumns
)

// 헤더 문구가 바뀌어도 덜 취약하도록 summary 속성 대신 첫 <th> 로 판별한다.
func kindOf(table *goquery.Selection) tableKind {
	switch strings.TrimSpace(table.Find("th").First().Text()) {
	case "계좌번호":
		return accountsTable
	case "상품명":
		return holdingsTable
	}
	return unknownTable
}

type row []string

func dataRows(table *goquery.Selection, columns int) []row {
	var rows []row
	table.Find("tr").Each(func(_ int, tr *goquery.Selection) {
		var cells row
		tr.Find("td").Each(func(_ int, td *goquery.Selection) {
			cells = append(cells, strings.TrimSpace(td.Text()))
		})
		if len(cells) == columns {
			rows = append(rows, cells)
		}
	})
	return rows
}

func (r row) toAccount() (domain.Account, error) {
	total, err := r.float(accTotalAsset, "자산총액")
	if err != nil {
		return domain.Account{}, err
	}
	withdrawable, err := r.float(accWithdrawable, "출금가능액")
	if err != nil {
		return domain.Account{}, err
	}

	return domain.Account{
		Number:        domain.NormalizeAccountNumber(r[accNumber]),
		DisplayNumber: r[accNumber],
		Type:          r[accType],
		TotalAsset:    total,
		Withdrawable:  withdrawable,
	}, nil
}

func (r row) toHolding(account string) (domain.Holding, error) {
	quantity, err := r.float(hdQuantity, "보유수량")
	if err != nil {
		return domain.Holding{}, err
	}
	evalAmount, err := r.float(hdEvalAmount, "평가금액")
	if err != nil {
		return domain.Holding{}, err
	}

	holding := domain.Holding{
		AccountNumber: account,
		Name:          r[hdName],
		Quantity:      quantity,
		EvalAmount:    evalAmount,
	}

	for _, field := range []struct {
		index int
		label string
		dst   **float64
	}{
		{hdCurrentPrice, "현재가", &holding.CurrentPrice},
		{hdAvgBuyPrice, "평균매입가", &holding.AvgBuyPrice},
		{hdBuyAmount, "매입금액", &holding.BuyAmount},
		{hdProfitLoss, "평가손익", &holding.ProfitLoss},
		{hdProfitRate, "손익률", &holding.ProfitRate},
	} {
		value, err := r.nullableFloat(field.index, field.label)
		if err != nil {
			return domain.Holding{}, err
		}
		*field.dst = value
	}

	return holding, nil
}

func (r row) float(index int, label string) (float64, error) {
	value, err := parseFloat(r[index])
	if err != nil {
		return 0, labeled(label, err)
	}
	return value, nil
}

func (r row) nullableFloat(index int, label string) (*float64, error) {
	value, err := parseNullableFloat(r[index])
	if err != nil {
		return nil, labeled(label, err)
	}
	return value, nil
}
