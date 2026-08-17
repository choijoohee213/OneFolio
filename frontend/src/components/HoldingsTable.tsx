import { won } from '../format'
import type { AccountSummary, Holding } from '../types'
import type { Quote } from '../liveQuotes'
import { HoldingRows } from './HoldingRows'

export type GroupMode = 'all' | 'account'

interface Props {
  holdings: Holding[]
  accounts: AccountSummary[]
  mode: GroupMode
  onModeChange: (mode: GroupMode) => void
  busy: boolean
  onAddHolding: () => void
  onScreenshot: () => void
  onEditHolding: (holding: Holding) => void
  showLive?: boolean
  onToggleLive?: () => void
  liveQuotesAvailable?: boolean
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
  onAddHolding,
  onScreenshot,
  onEditHolding,
  showLive,
  onToggleLive,
  liveQuotesAvailable,
  showUSD,
  onToggleUSD,
  quotes,
  usdKrw,
}: Props) {
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
          <button type="button" className="add-toggle" disabled={busy} onClick={onAddHolding}>
            종목 추가
          </button>
          <button type="button" className="add-toggle" disabled={busy} onClick={onScreenshot}>
            스크린샷
          </button>
          {onToggleLive && (
            <button type="button" className="add-toggle" disabled={busy} aria-pressed={showLive} onClick={onToggleLive}>
              실시간 시세
            </button>
          )}
          {liveQuotesAvailable && (
            <button type="button" className="add-toggle" aria-pressed={showUSD} onClick={onToggleUSD}>
              달러로 보기
            </button>
          )}
        </div>
      </header>

      {mode === 'all' ? (
        <HoldingRows
          holdings={mergeByName(holdings)}
          busy={busy}
          onEdit={onEditHolding}
          showUSD={showUSD}
          quotes={quotes}
          usdKrw={usdKrw}
        />
      ) : (
        <AccountGroups
          holdings={holdings}
          accounts={accounts}
          busy={busy}
          onEdit={onEditHolding}
          showUSD={showUSD}
          quotes={quotes}
          usdKrw={usdKrw}
        />
      )}
    </section>
  )
}

function AccountGroups({
  holdings,
  accounts,
  busy,
  onEdit,
  showUSD,
  quotes,
  usdKrw,
}: {
  holdings: Holding[]
  accounts: AccountSummary[]
  busy: boolean
  onEdit: (holding: Holding) => void
  showUSD?: boolean
  quotes?: Record<string, Quote> | null
  usdKrw?: number | null
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
          <HoldingRows
            holdings={rows}
            busy={busy}
            onEdit={onEdit}
            showUSD={showUSD}
            quotes={quotes}
            usdKrw={usdKrw}
          />
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
