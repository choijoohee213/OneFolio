import { useState } from 'react'
import { won } from '../format'
import type { AccountSummary, Holding } from '../types'
import type { Quote } from '../liveQuotes'
import { HoldingRows, type ViewMode } from './HoldingRows'

export type GroupMode = 'all' | 'account'

const VIEW_KEY = 'onefolio:holdings-view'

// 좁은 화면에서는 카드가, 넓은 화면에서는 표가 편하다. 한 번 고르면 그 선택을
// 기억한다 — 화면 폭은 첫 기본값을 정할 때만 본다.
function loadView(): ViewMode {
  const saved = localStorage.getItem(VIEW_KEY)
  if (saved === 'card' || saved === 'table') return saved
  return window.matchMedia('(max-width: 34rem)').matches ? 'card' : 'table'
}

interface Props {
  holdings: Holding[]
  accounts: AccountSummary[]
  mode: GroupMode
  onModeChange: (mode: GroupMode) => void
  busy: boolean
  onEditHolding: (holding: Holding) => void
  showLive?: boolean
  showUSD?: boolean
  onToggleUSD?: () => void
  quotes?: Record<string, Quote> | null
  usdKrw?: number | null
}

export function HoldingsTable({
  holdings,
  accounts,
  mode,
  onModeChange,
  busy,
  onEditHolding,
  showLive,
  showUSD,
  onToggleUSD,
  quotes,
  usdKrw,
}: Props) {
  const [view, setView] = useState<ViewMode>(loadView)
  const [detail, setDetail] = useState<Holding | null>(null)

  function changeView(next: ViewMode) {
    setView(next)
    localStorage.setItem(VIEW_KEY, next)
  }

  const rowProps = {
    busy,
    onEdit: onEditHolding,
    showUSD,
    showLive,
    quotes,
    usdKrw,
    view,
    detail,
    onOpenDetail: setDetail,
    onCloseDetail: () => setDetail(null),
  }

  return (
    <section className="holdings">
      <header className="section-head">
        <h2>보유 종목</h2>
        <div className="head-actions">
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
          <div className="toggle" role="group" aria-label="표시 통화">
            <button type="button" aria-pressed={!showUSD} onClick={() => showUSD && onToggleUSD?.()}>
              원화
            </button>
            <button type="button" aria-pressed={!!showUSD} onClick={() => !showUSD && onToggleUSD?.()}>
              달러
            </button>
          </div>
          <div className="toggle" role="group" aria-label="목록 모양">
            <button type="button" aria-pressed={view === 'card'} onClick={() => changeView('card')}>
              카드
            </button>
            <button type="button" aria-pressed={view === 'table'} onClick={() => changeView('table')}>
              표
            </button>
          </div>
        </div>
      </header>

      {mode === 'all' ? (
        <HoldingRows holdings={mergeByName(holdings)} {...rowProps} />
      ) : (
        <AccountGroups holdings={holdings} accounts={accounts} rowProps={rowProps} />
      )}
    </section>
  )
}

function AccountGroups({
  holdings,
  accounts,
  rowProps,
}: {
  holdings: Holding[]
  accounts: AccountSummary[]
  rowProps: Omit<React.ComponentProps<typeof HoldingRows>, 'holdings'>
}) {
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
          <HoldingRows holdings={rows} {...rowProps} />
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
      // accountNumber 는 그대로 첫 항목 것을 쓴다. 화면에서는 name 으로만 묶어
      // 보여주므로 값 자체는 의미가 없지만, manual: 접두사가 남아 있어야
      // 직접 추가한 자산인지 구분하는 로직(수량 표시 등)이 병합 후에도 맞는다.
      merged.set(holding.name, { ...holding })
      continue
    }
    found.quantity += holding.quantity
    found.evalAmount += holding.evalAmount
    found.weight += holding.weight
    found.buyAmount = addNullable(found.buyAmount, holding.buyAmount)
    found.profitLoss = addNullable(found.profitLoss, holding.profitLoss)
    // 여러 계좌가 합쳐진 행은 어느 계좌를 고치는 건지 알 수 없다. 수정을 막으려고
    // 표시해 둔다 — 계좌별 보기에서 계좌를 특정해 고쳐야 한다.
    found.mergedFromMultipleAccounts = true
  }

  for (const holding of merged.values()) {
    if (!holding.mergedFromMultipleAccounts) continue
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
