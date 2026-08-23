import { categoryColor, percent, quantity, signedPercent, signedUsd, signedWon, usd, won } from '../format'
import type { Holding } from '../types'
import { isManualHolding } from '../types'
import type { Quote } from '../liveQuotes'
import { Modal } from './Modal'

interface Props {
  holdings: Holding[]
  busy: boolean
  onEdit: (holding: Holding) => void
  showUSD?: boolean
  showLive?: boolean
  quotes?: Record<string, Quote> | null
  usdKrw?: number | null
  detail: Holding | null
  onOpenDetail: (holding: Holding) => void
  onCloseDetail: () => void
}

// 목록에는 종목명·수량·평가금액·평가손익만 둔다. 한 줄에 다 넣으려고 글자를
// 줄이면 정작 크게 봐야 할 금액이 작아진다 — 나머지는 눌러서 본다.
export function HoldingCards({
  holdings,
  busy,
  onEdit,
  showUSD,
  showLive,
  quotes,
  usdKrw,
  detail,
  onOpenDetail,
  onCloseDetail,
}: Props) {
  const view = (holding: Holding) => amounts(holding, showUSD, quotes, usdKrw)

  return (
    <>
      <ul className="holding-cards">
        {holdings.map((holding) => {
          const { amount, signedAmount } = view(holding)
          return (
            <li key={`${holding.accountNumber}-${holding.name}`}>
              <button type="button" className="holding-card" onClick={() => onOpenDetail(holding)}>
                <span className="hc-main">
                  <span className="hc-name">
                    <span className="swatch small" style={{ background: categoryColor(holding.category) }} />
                    {holding.name}
                  </span>
                  <span className="hc-sub">
                    {isManualHolding(holding) && holding.quantity === 0
                      ? holding.category
                      : `${quantity(holding.quantity)}주`}
                  </span>
                </span>
                <span className="hc-figures">
                  <span className="hc-eval">{amount(holding.evalAmount)}</span>
                  <span className={`hc-pl ${sign(holding.profitLoss)}`}>
                    {holding.profitLoss === null
                      ? '—'
                      : `${signedAmount(holding.profitLoss)}${
                          holding.profitRate === null ? '' : ` (${signedPercent(holding.profitRate)})`
                        }`}
                  </span>
                </span>
              </button>
            </li>
          )
        })}
      </ul>

      <Modal open={detail !== null} title={detail?.name ?? ''} onClose={onCloseDetail}>
        {detail && (
          <HoldingDetail
            holding={detail}
            busy={busy}
            showLive={showLive}
            {...view(detail)}
            currentPrice={currentPriceOf(detail, showUSD, quotes, usdKrw)}
            onEdit={() => {
              onCloseDetail()
              onEdit(detail)
            }}
          />
        )}
      </Modal>
    </>
  )
}

function HoldingDetail({
  holding,
  busy,
  showLive,
  amount,
  signedAmount,
  currentPrice,
  onEdit,
}: {
  holding: Holding
  busy: boolean
  showLive?: boolean
  amount: (value: number) => string
  signedAmount: (value: number) => string
  currentPrice: string | null
  onEdit: () => void
}) {
  const buyAmount =
    holding.buyAmount ??
    (holding.profitLoss === null ? null : holding.evalAmount - holding.profitLoss)

  return (
    <div className="holding-detail">
      <p className="hd-caption">
        <span className="swatch small" style={{ background: categoryColor(holding.category) }} />
        {holding.category}
        {holding.code && <span className="hd-code">{holding.code}</span>}
      </p>

      <dl>
        <Row label="평가금액" value={amount(holding.evalAmount)} strong />
        <Row label="매입금액" value={buyAmount === null ? '—' : amount(buyAmount)} />
        <Row
          label="평가손익"
          value={
            holding.profitLoss === null
              ? '—'
              : `${signedAmount(holding.profitLoss)}${
                  holding.profitRate === null ? '' : ` (${signedPercent(holding.profitRate)})`
                }`
          }
          tone={sign(holding.profitLoss)}
        />
        <Row
          label="보유수량"
          value={
            isManualHolding(holding) && holding.quantity === 0 ? '—' : `${quantity(holding.quantity)}주`
          }
        />
        <Row label="평균단가" value={holding.avgBuyPrice === null ? '—' : amount(holding.avgBuyPrice)} />
        {showLive && <Row label="현재가" value={currentPrice ?? '—'} />}
        <Row label="비중" value={percent(holding.weight)} />
      </dl>

      <button
        type="button"
        className="hd-edit"
        disabled={busy || holding.mergedFromMultipleAccounts}
        title={
          holding.mergedFromMultipleAccounts
            ? '여러 계좌에 나뉘어 있습니다. 계좌별 보기에서 고쳐주세요'
            : undefined
        }
        onClick={onEdit}
      >
        수정
      </button>
    </div>
  )
}

function Row({
  label,
  value,
  strong,
  tone,
}: {
  label: string
  value: string
  strong?: boolean
  tone?: string
}) {
  return (
    <>
      <dt>{label}</dt>
      <dd className={`${strong ? 'strong ' : ''}${tone ?? ''}`}>{value}</dd>
    </>
  )
}

// 달러 보기일 때 원화 값을 나눠 쓰는 규칙은 표와 같아야 한다.
function amounts(
  holding: Holding,
  showUSD: boolean | undefined,
  quotes: Record<string, Quote> | null | undefined,
  usdKrw: number | null | undefined,
) {
  const quote = holding.code ? quotes?.[holding.code] : undefined
  const fx = showUSD && quote?.currency === 'USD' ? usdKrw ?? null : null
  return {
    amount: (value: number) => (fx ? usd(value / fx) : won(value)),
    signedAmount: (value: number) => (fx ? signedUsd(value / fx) : signedWon(value)),
  }
}

function currentPriceOf(
  holding: Holding,
  showUSD: boolean | undefined,
  quotes: Record<string, Quote> | null | undefined,
  usdKrw: number | null | undefined,
): string | null {
  const quote = holding.code ? quotes?.[holding.code] : undefined
  if (!quote) return null
  const fx = showUSD && quote.currency === 'USD' ? usdKrw ?? null : null
  if (quote.currency === 'USD') return fx ? usd(quote.price) : won(quote.price * (usdKrw ?? 0))
  return won(quote.price)
}

function sign(value: number | null): string {
  if (value === null || value === 0) return ''
  return value > 0 ? 'gain' : 'loss'
}
