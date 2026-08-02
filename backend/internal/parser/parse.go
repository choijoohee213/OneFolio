// Package parser 는 미래에셋 "계좌별잔고" 파일을 읽는다.
// 확장자는 .xls 지만 실제 내용은 UTF-8 HTML 이므로 엑셀이 아닌 HTML 로 파싱한다.
package parser

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/choijoohee213/OneFolio/backend/internal/domain"
)

type Result struct {
	Accounts []domain.Account
	Holdings []domain.Holding
	Cash     []domain.Holding
}

var ErrNotBalanceFile = errors.New("전체 계좌현황 테이블이 없음: 잔고파일이 아닙니다")

func Parse(r io.Reader) (*Result, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("HTML 파싱 실패: %w", err)
	}

	var file balanceFile
	if err := file.read(doc); err != nil {
		return nil, err
	}
	if len(file.result.Accounts) == 0 {
		return nil, ErrNotBalanceFile
	}
	return &file.result, nil
}

// balanceFile 은 테이블을 문서 순서대로 훑는다. 상품보유현황 테이블에는 계좌번호가
// 없고 바로 앞 테이블 헤더에만 있어서, 마지막으로 본 계좌를 들고 다녀야 한다.
type balanceFile struct {
	result  Result
	account string
}

func (f *balanceFile) read(doc *goquery.Document) error {
	var err error
	doc.Find("table").EachWithBreak(func(_ int, table *goquery.Selection) bool {
		err = f.readTable(table)
		return err == nil
	})
	return err
}

func (f *balanceFile) readTable(table *goquery.Selection) error {
	if account, ok := holdingsOwner(table); ok {
		f.account = account
		return nil
	}

	switch kindOf(table) {
	case accountsTable:
		return f.readAccounts(table)
	case holdingsTable:
		return f.readHoldings(table)
	}
	return nil
}

func (f *balanceFile) readAccounts(table *goquery.Selection) error {
	for _, r := range dataRows(table, accColumns) {
		account, err := r.toAccount()
		if err != nil {
			return fmt.Errorf("계좌 %s: %w", r[accNumber], err)
		}
		f.result.Accounts = append(f.result.Accounts, account)
	}
	return nil
}

func (f *balanceFile) readHoldings(table *goquery.Selection) error {
	if f.account == "" {
		return errors.New("상품보유현황 테이블 앞에 계좌번호 헤더가 없음")
	}

	for _, r := range dataRows(table, hdColumns) {
		holding, err := r.toHolding(f.account)
		if err != nil {
			return fmt.Errorf("%s: %w", r[hdName], err)
		}
		if isCash(holding) {
			f.result.Cash = append(f.result.Cash, holding)
			continue
		}
		f.result.Holdings = append(f.result.Holdings, holding)
	}
	return nil
}

// isCash 는 취득원가가 없는 행(미국달러 등)을 매매 종목이 아니라고 본다.
// 종합계좌 파일 하나만 보고 세운 규칙이라, ISA·연금저축 파일에서 예수금 행의
// 모양이 다르면 조정해야 한다.
func isCash(h domain.Holding) bool {
	return h.AvgBuyPrice == nil && h.ProfitRate == nil
}

var holdingsHeader = regexp.MustCompile(`\[([\d-]+)\]\s*상품보유현황`)

func holdingsOwner(table *goquery.Selection) (string, bool) {
	m := holdingsHeader.FindStringSubmatch(table.Text())
	if m == nil {
		return "", false
	}
	return normalizeAccountNo(m[1]), true
}

// normalizeAccountNo 는 숫자만 남긴다. 같은 계좌라도 전체 계좌현황은
// "616-8228-7204-0", 상품보유현황 헤더는 "616-82-2872040" 이라 그대로는 join 이 안 된다.
func normalizeAccountNo(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
