import { won } from '../format'
import type { AccountSummary, Holding } from '../types'
import { HoldingRows } from './HoldingRows'

export type GroupMode = 'all' | 'account'

interface Props {
  holdings: Holding[]
  accounts: AccountSummary[]
  mode: GroupMode
  onModeChange: (mode: GroupMode) => void
}

export function HoldingsTable({ holdings, accounts, mode, onModeChange }: Props) {
  return (
    <section className="holdings">
      <header className="section-head">
        <h2>보유 종목</h2>
        <div className="toggle" role="group" aria-label="보기 방식">
          <button type="button" aria-pressed={mode === 'all'} onClick={() => onModeChange('all')}>
            전체
          </button>
          <button
            type="button"
            aria-pressed={mode === 'account'}
            onClick={() => onModeChange('account')}
          >
            계좌별
          </button>
        </div>
      </header>

      {mode === 'all' ? (
        <HoldingRows holdings={mergeByName(holdings)} />
      ) : (
        <AccountGroups holdings={holdings} accounts={accounts} />
      )}
    </section>
  )
}

function AccountGroups({ holdings, accounts }: { holdings: Holding[]; accounts: AccountSummary[] }) {
  const groups = accounts
    .map((account) => ({
      account,
      rows: holdings
        .filter((holding) => holding.accountNumber === account.number)
        .sort((a, b) => b.evalAmount - a.evalAmount),
    }))
    .filter((group) => group.rows.length > 0)

  return (
    <div className="groups">
      {groups.map(({ account, rows }) => (
        <details key={account.number} className="group">
          <summary>
            <span className="chevron" aria-hidden="true">
              ▶
            </span>
            <span className="group-name">{account.type}</span>
            <span className="group-count">{rows.length}종목</span>
            <span className="group-amount">{won(sumEvalAmount(rows))}</span>
          </summary>
          <HoldingRows holdings={rows} />
        </details>
      ))}
    </div>
  )
}

function sumEvalAmount(holdings: Holding[]): number {
  return holdings.reduce((total, holding) => total + holding.evalAmount, 0)
}

// 같은 종목을 여러 계좌에 나눠 들고 있으면 API 는 계좌마다 별도 행으로 준다.
// 전체 보기에서는 수량·금액을 합치고, 평단과 손익률은 합산 매입금액에서 다시 낸다.
function mergeByName(holdings: Holding[]): Holding[] {
  const merged = new Map<string, Holding>()

  for (const holding of holdings) {
    const found = merged.get(holding.name)
    if (!found) {
      merged.set(holding.name, { ...holding, accountNumber: 'merged' })
      continue
    }
    found.quantity += holding.quantity
    found.evalAmount += holding.evalAmount
    found.weight += holding.weight
    found.buyAmount = addNullable(found.buyAmount, holding.buyAmount)
    found.profitLoss = addNullable(found.profitLoss, holding.profitLoss)
  }

  for (const holding of merged.values()) {
    holding.avgBuyPrice = holding.buyAmount === null ? null : holding.buyAmount / holding.quantity
    holding.profitRate =
      holding.buyAmount === null || holding.buyAmount === 0 || holding.profitLoss === null
        ? null
        : (holding.profitLoss / holding.buyAmount) * 100
  }

  return [...merged.values()].sort((a, b) => b.evalAmount - a.evalAmount)
}

function addNullable(a: number | null, b: number | null): number | null {
  if (a === null || b === null) return null
  return a + b
}
