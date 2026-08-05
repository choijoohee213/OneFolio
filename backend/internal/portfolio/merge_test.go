package portfolio

import (
	"testing"

	"github.com/choijoohee213/OneFolio/backend/internal/domain"
	"github.com/choijoohee213/OneFolio/backend/internal/parser"
)

func account(number string, total float64) domain.Account {
	return domain.Account{Number: number, TotalAsset: total}
}

func holding(accountNo, name string, evalAmount float64) domain.Holding {
	return domain.Holding{AccountNumber: accountNo, Name: name, EvalAmount: evalAmount}
}

// 어느 파일에나 계좌현황 전체가 들어있으므로 계좌는 한 번씩만 남아야 한다.
func TestMergeDeduplicatesAccounts(t *testing.T) {
	all := []domain.Account{account("111", 1000), account("222", 2000)}
	merged := Merge(
		&parser.Result{Accounts: all, Holdings: []domain.Holding{holding("111", "삼성전자", 500)}},
		&parser.Result{Accounts: all, Holdings: []domain.Holding{holding("222", "AMD", 700)}},
	)

	if len(merged.Accounts) != 2 {
		t.Fatalf("계좌 %d개, want 2", len(merged.Accounts))
	}
	if len(merged.Holdings) != 2 {
		t.Fatalf("종목 %d개, want 2", len(merged.Holdings))
	}
}

// 같은 파일을 두 번 올려도 종목이 불어나면 안 된다.
func TestMergeSameFileTwice(t *testing.T) {
	file := &parser.Result{
		Accounts: []domain.Account{account("111", 1000)},
		Holdings: []domain.Holding{holding("111", "삼성전자", 500)},
		Cash:     []domain.Holding{holding("111", "미국달러", -100)},
	}

	merged := Merge(file, file)
	if len(merged.Holdings) != 1 || len(merged.Cash) != 1 {
		t.Fatalf("종목 %d개 / 현금성 %d개, want 1 / 1", len(merged.Holdings), len(merged.Cash))
	}
}

// 계좌가 다르면 같은 종목명이라도 별개로 센다.
func TestMergeKeepsSameNameInDifferentAccounts(t *testing.T) {
	merged := Merge(
		&parser.Result{Accounts: []domain.Account{account("111", 1000)}, Holdings: []domain.Holding{holding("111", "TIGER 미국S&P500", 500)}},
		&parser.Result{Accounts: []domain.Account{account("222", 2000)}, Holdings: []domain.Holding{holding("222", "TIGER 미국S&P500", 700)}},
	)

	if len(merged.Holdings) != 2 {
		t.Fatalf("종목 %d개, want 2", len(merged.Holdings))
	}
}

// 파일마다 어느 계좌를 담당했는지 알아야 클라이언트가 계좌 단위로 파일을 지운다.
func TestCoveredAccounts(t *testing.T) {
	result := &parser.Result{
		Accounts: []domain.Account{account("111", 1000), account("222", 2000), account("333", 3000)},
		Holdings: []domain.Holding{holding("222", "삼성전자", 500), holding("222", "AMD", 300)},
		Cash:     []domain.Holding{holding("222", "미국달러", -100)},
	}

	got := CoveredAccounts(result)
	if len(got) != 1 || got[0] != "222" {
		t.Errorf("CoveredAccounts = %v, want [222] (계좌 요약이 아니라 종목 기준)", got)
	}
}

func TestMergeLatestValueWins(t *testing.T) {
	merged := Merge(
		&parser.Result{Accounts: []domain.Account{account("111", 1000)}, Holdings: []domain.Holding{holding("111", "삼성전자", 500)}},
		&parser.Result{Accounts: []domain.Account{account("111", 1000)}, Holdings: []domain.Holding{holding("111", "삼성전자", 900)}},
	)

	if len(merged.Holdings) != 1 || merged.Holdings[0].EvalAmount != 900 {
		t.Fatalf("Holdings = %+v, want 평가금액 900 하나", merged.Holdings)
	}
}
