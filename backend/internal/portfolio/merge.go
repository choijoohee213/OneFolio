// Package portfolio 는 계좌별 잔고파일을 하나로 합치고 자산배분을 집계한다.
package portfolio

import (
	"github.com/choijoohee213/OneFolio/backend/internal/domain"
	"github.com/choijoohee213/OneFolio/backend/internal/parser"
)

type Portfolio struct {
	Accounts []domain.Account
	Holdings []domain.Holding
	Cash     []domain.Holding
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
