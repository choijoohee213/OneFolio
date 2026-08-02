// Package parser 는 미래에셋 "계좌별잔고" 파일을 파싱한다.
//
// 확장자는 .xls 지만 실제 내용은 UTF-8 HTML 문서다. 엑셀 라이브러리가 아니라
// HTML 테이블 파서로 읽어야 한다.
//
// 파일 구조:
//
//	<table class="col_type" summary="계좌번호, ...">   전체 계좌현황
//	<table><td colspan=7>[616-82-2872040] 상품보유현황</td></table>
//	<table class="col_type" summary="상품명, ...">      위 계좌의 종목 상세
//
// 상품보유현황은 다운로드 시 선택한 계좌만 나오므로, 계좌 3개를 각각 받아
// 파일별로 Parse 한 뒤 합쳐야 전체 자산이 된다.
package parser

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/choijoohee213/OneFolio/backend/internal/domain"
)

// Result 는 잔고파일 하나를 파싱한 결과다.
type Result struct {
	Accounts []domain.Account // 전체 계좌현황 (파일마다 계좌 전부가 들어있다)
	Holdings []domain.Holding // 상품보유현황 중 매매 종목 (다운로드 시 선택한 계좌 것만)

	// Cash 는 상품보유현황에 같이 들어있지만 종목이 아닌 행이다 (미국달러 등).
	// 종목 목록·손익 계산에서 빼되, 나중에 현금 비중을 넣을 때 쓰려고 버리지 않고 담아둔다.
	Cash []domain.Holding
}

// 상품보유현황 헤더에서 계좌번호를 뽑는다. 예: "[616-82-2872040] 상품보유현황"
var holdingsHeaderRe = regexp.MustCompile(`\[([\d-]+)\]\s*상품보유현황`)

// Parse 는 잔고파일 HTML 을 읽어 계좌 목록과 보유종목 목록을 돌려준다.
func Parse(r io.Reader) (*Result, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("HTML 파싱 실패: %w", err)
	}

	res := &Result{}

	// 상품보유현황 테이블에는 계좌번호가 없다. 바로 앞 테이블의 헤더에만 있으므로
	// 문서 순서대로 훑으면서 마지막에 본 계좌번호를 기억해 둔다.
	var currentAccount string
	var parseErr error

	doc.Find("table").EachWithBreak(func(_ int, table *goquery.Selection) bool {
		if m := holdingsHeaderRe.FindStringSubmatch(table.Text()); m != nil {
			currentAccount = normalizeAccountNo(m[1])
			return true
		}

		switch tableKind(table) {
		case kindAccounts:
			accounts, err := parseAccounts(table)
			if err != nil {
				parseErr = err
				return false
			}
			res.Accounts = append(res.Accounts, accounts...)
		case kindHoldings:
			if currentAccount == "" {
				parseErr = fmt.Errorf("상품보유현황 테이블보다 앞에 계좌번호 헤더가 없음")
				return false
			}
			rows, err := parseHoldings(table, currentAccount)
			if err != nil {
				parseErr = err
				return false
			}
			for _, h := range rows {
				if isCashRow(h) {
					res.Cash = append(res.Cash, h)
					continue
				}
				res.Holdings = append(res.Holdings, h)
			}
		}
		return true
	})

	if parseErr != nil {
		return nil, parseErr
	}
	if len(res.Accounts) == 0 {
		return nil, fmt.Errorf("전체 계좌현황 테이블을 찾지 못함 (잔고파일이 맞는지 확인 필요)")
	}
	return res, nil
}

type kind int

const (
	kindUnknown kind = iota
	kindAccounts
	kindHoldings
)

// tableKind 는 헤더 행의 첫 <th> 로 테이블 종류를 판별한다.
// summary 속성 대신 헤더 텍스트를 쓰는 편이 문구 변경에 덜 취약하다.
func tableKind(table *goquery.Selection) kind {
	switch strings.TrimSpace(table.Find("th").First().Text()) {
	case "계좌번호":
		return kindAccounts
	case "상품명":
		return kindHoldings
	}
	return kindUnknown
}

// 전체 계좌현황 컬럼: 계좌번호 | 계좌유형 | 계좌별명 | 자산총액 | 출금가능액 | 바로가기
const accountCols = 6

func parseAccounts(table *goquery.Selection) ([]domain.Account, error) {
	var accounts []domain.Account
	var err error

	table.Find("tr").EachWithBreak(func(_ int, tr *goquery.Selection) bool {
		cells := cellTexts(tr)
		if len(cells) != accountCols {
			return true // 헤더 행(<th>)이나 예상 밖의 행은 건너뛴다
		}

		total, e := parseFloat(cells[3])
		if e != nil {
			err = fmt.Errorf("계좌 %s 자산총액: %w", cells[0], e)
			return false
		}
		withdrawable, e := parseFloat(cells[4])
		if e != nil {
			err = fmt.Errorf("계좌 %s 출금가능액: %w", cells[0], e)
			return false
		}

		accounts = append(accounts, domain.Account{
			Number:        normalizeAccountNo(cells[0]),
			DisplayNumber: cells[0],
			Type:          cells[1],
			TotalAsset:    total,
			Withdrawable:  withdrawable,
		})
		return true
	})

	return accounts, err
}

// 상품보유현황 컬럼: 상품명 | 보유수량 | 현재가 | 평균매입가 | 매입금액 | 평가금액 | 평가손익 | 손익률
const holdingCols = 8

func parseHoldings(table *goquery.Selection, accountNo string) ([]domain.Holding, error) {
	var holdings []domain.Holding
	var err error

	table.Find("tr").EachWithBreak(func(_ int, tr *goquery.Selection) bool {
		cells := cellTexts(tr)
		if len(cells) != holdingCols {
			return true
		}

		name := cells[0]
		fail := func(col string, e error) bool {
			err = fmt.Errorf("%s %s: %w", name, col, e)
			return false
		}

		// 외화 마이너스 잔고처럼 수량·평가금액이 음수인 행이 있다. 값은 항상 존재한다.
		quantity, e := parseFloat(cells[1])
		if e != nil {
			return fail("보유수량", e)
		}
		evalAmount, e := parseFloat(cells[5])
		if e != nil {
			return fail("평가금액", e)
		}

		// 현금성 자산은 평단·손익 관련 값이 "-" 로 비어 있다.
		currentPrice, e := parseNullableFloat(cells[2])
		if e != nil {
			return fail("현재가", e)
		}
		avgBuyPrice, e := parseNullableFloat(cells[3])
		if e != nil {
			return fail("평균매입가", e)
		}
		buyAmount, e := parseNullableFloat(cells[4])
		if e != nil {
			return fail("매입금액", e)
		}
		profitLoss, e := parseNullableFloat(cells[6])
		if e != nil {
			return fail("평가손익", e)
		}
		profitRate, e := parseNullableFloat(cells[7])
		if e != nil {
			return fail("손익률", e)
		}

		holdings = append(holdings, domain.Holding{
			AccountNumber: accountNo,
			Name:          name,
			Quantity:      quantity,
			CurrentPrice:  currentPrice,
			AvgBuyPrice:   avgBuyPrice,
			BuyAmount:     buyAmount,
			EvalAmount:    evalAmount,
			ProfitLoss:    profitLoss,
			ProfitRate:    profitRate,
		})
		return true
	})

	return holdings, err
}

// isCashRow 는 매매 종목이 아닌 행(미국달러 등)인지 판별한다.
// 취득원가가 없으면 매수한 포지션이 아니라고 본다. 상품명 목록을 하드코딩하는
// 것보다 낫지만, 지금은 종합계좌 파일 하나만 확인한 규칙이다. ISA·연금저축
// 파일에서 예수금 행의 모양이 다르면 여기를 조정해야 한다.
func isCashRow(h domain.Holding) bool {
	return h.AvgBuyPrice == nil && h.ProfitRate == nil
}

func cellTexts(tr *goquery.Selection) []string {
	var out []string
	tr.Find("td").Each(func(_ int, td *goquery.Selection) {
		out = append(out, strings.TrimSpace(td.Text()))
	})
	return out
}

// normalizeAccountNo 는 계좌번호에서 숫자만 남긴다.
// 같은 계좌라도 전체 계좌현황은 "616-8228-7204-0", 상품보유현황 헤더는
// "616-82-2872040" 으로 하이픈 위치가 달라서, 그대로 두면 join 이 안 된다.
func normalizeAccountNo(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
