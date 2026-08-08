// Package portfolio 는 계좌별 잔고파일을 하나로 합치고 자산배분을 집계한다.
package portfolio

import (
	"strings"

	"github.com/choijoohee213/OneFolio/backend/internal/domain"
	"github.com/choijoohee213/OneFolio/backend/internal/parser"
)

type Portfolio struct {
	Accounts []domain.Account
	Holdings []domain.Holding
	Cash     []domain.Holding
}

// ManualAccountPrefix 는 잔고파일에 없는, 사용자가 직접 추가한 보유를 표시하는
// 가상 계좌번호 접두사다. 이런 보유는 어느 계좌 자산총액에도 기대지 않고
// Summarize 에서 자기 평가금액만큼 스스로 집계 분모를 채운다.
const ManualAccountPrefix = "manual:"

func IsManual(accountNumber string) bool {
	return strings.HasPrefix(accountNumber, ManualAccountPrefix)
}

// CoveredAccounts 는 이 파일이 종목 상세를 담고 있는 계좌 번호를 돌려준다.
// 잔고파일은 어느 것을 받아도 전체 계좌현황을 담고 있어서, 계좌 목록만으로는
// 무엇을 담당한 파일인지 알 수 없다. 클라이언트가 계좌 단위로 파일을 관리할 때 쓴다.
func CoveredAccounts(result *parser.Result) []string {
	seen := make(map[string]bool)
	var numbers []string
	for _, group := range [][]domain.Holding{result.Holdings, result.Cash} {
		for _, h := range group {
			if seen[h.AccountNumber] {
				continue
			}
			seen[h.AccountNumber] = true
			numbers = append(numbers, h.AccountNumber)
		}
	}
	return numbers
}

// Merge 는 계좌별로 받은 파일들을 합친다. 어느 파일에나 전체 계좌현황이 똑같이
// 들어있고 같은 파일을 두 번 올릴 수도 있어서, 계좌와 종목 모두 중복을 제거한다.
func Merge(files ...*parser.Result) *Portfolio {
	merged := &Portfolio{}

	seenAccount := make(map[string]bool)
	for _, file := range files {
		for _, account := range file.Accounts {
			if seenAccount[account.Number] {
				continue
			}
			seenAccount[account.Number] = true
			merged.Accounts = append(merged.Accounts, account)
		}
	}

	holdings := newHoldingSet()
	cash := newHoldingSet()
	for _, file := range files {
		holdings.addAll(file.Holdings)
		cash.addAll(file.Cash)
	}
	merged.Holdings = holdings.items
	merged.Cash = cash.items

	return merged
}

// 같은 계좌·같은 종목은 나중 파일 값으로 덮어쓰되 순서는 유지한다.
type holdingSet struct {
	items []domain.Holding
	index map[string]int
}

func newHoldingSet() *holdingSet {
	return &holdingSet{index: make(map[string]int)}
}

func (s *holdingSet) addAll(holdings []domain.Holding) {
	for _, h := range holdings {
		key := h.AccountNumber + "\x00" + h.Name
		if i, ok := s.index[key]; ok {
			s.items[i] = h
			continue
		}
		s.index[key] = len(s.items)
		s.items = append(s.items, h)
	}
}
