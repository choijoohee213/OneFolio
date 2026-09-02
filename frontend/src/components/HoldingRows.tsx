import { useEffect, useRef, useState } from 'react'
import { categoryColor, percent, quantity, signedPercent, signedUsd, signedWon, usd, won } from '../format'
import type { Holding } from '../types'
import { isManualHolding } from '../types'
import type { Quote } from '../liveQuotes'
import { usdFigures } from '../usdView'
import { HoldingCards } from './HoldingCards'

export type ViewMode = 'card' | 'table'

interface Props {
  holdings: Holding[]
  busy: boolean
  onEdit: (holding: Holding) => void
  showUSD?: boolean
  showLive?: boolean
  quotes?: Record<string, Quote> | null
  usdKrw?: number | null
  view: ViewMode
  detail: Holding | null
  onOpenDetail: (holding: Holding) => void
  onCloseDetail: () => void
}

export function HoldingRows(props: Props) {
  return props.view === 'card' ? <HoldingCards {...props} /> : <HoldingTable {...props} />
}

function HoldingTable({ holdings, busy, onEdit, showUSD, showLive, quotes, usdKrw }: Props) {
  return (
    <div className="scroll">
      <table>
        <thead>
          <tr>
            <th>종목</th>
            <th>분류</th>
            <th className="num">수량</th>
            <th className="num">평단가</th>
            {showLive && <th className="num">현재가</th>}
            <th className="num">평가금액</th>
            <th className="num">평가손익</th>
            <th className="num">손익률</th>
            <th className="num">비중</th>
            <th />
          </tr>
        </thead>
        <tbody>
          {holdings.map((holding) => {
            const quote = holding.code ? quotes?.[holding.code] : undefined
            // 달러 보기는 원화 값을 나누지 않고 달러끼리 계산한다.
            const inUsd = showUSD ? usdFigures(holding, quote, usdKrw, showLive) : null
            const view = inUsd ?? holding
            const amount = (v: number) => (inUsd ? usd(v) : won(v))
            const signedAmount = (v: number) => (inUsd ? signedUsd(v) : signedWon(v))
            const currentPrice = () => {
              if (!quote) return '—'
              if (inUsd) return usd(inUsd.currentPrice ?? quote.price)
              if (quote.currency === 'USD') return won(quote.price * (usdKrw ?? 0))
              return won(quote.price)
            }

            return (
              <tr key={`${holding.accountNumber}-${holding.name}`}>
                <td className="name">
                  {holding.name}
                  {holding.original && (
                    <span className="edited-tag" title="잔고파일 값을 직접 고친 종목">
                      수정됨
                    </span>
                  )}
                </td>
                <td>
                  <span className="swatch small" style={{ background: categoryColor(holding.category) }} />
                  {holding.category}
                </td>
                <td className="num">
                  {isManualHolding(holding) && holding.quantity === 0 ? '—' : quantity(holding.quantity)}
                </td>
                <td className="num">{view.avgBuyPrice === null ? '—' : amount(view.avgBuyPrice)}</td>
                {showLive && (
                  <PriceCell
                    quotePrice={quote?.price ?? null}
                    prevClose={quote?.prevClose ?? null}
                    formatted={currentPrice()}
                  />
                )}
                <td className="num">{amount(view.evalAmount)}</td>
                <td className={`num ${sign(view.profitLoss)}`}>
                  {view.profitLoss === null ? '—' : signedAmount(view.profitLoss)}
                </td>
                <td className={`num ${sign(view.profitRate)}`}>
                  {view.profitRate === null ? '—' : signedPercent(view.profitRate)}
                </td>
                <td className="num">{percent(holding.weight)}</td>
                <td className="num">
                  <button
                    type="button"
                    className="link"
                    disabled={busy || holding.mergedFromMultipleAccounts}
                    title={
                      holding.mergedFromMultipleAccounts
                        ? '여러 계좌에 나뉘어 있습니다. 계좌별 보기에서 고쳐주세요'
                        : undefined
                    }
                    onClick={() => onEdit(holding)}
                    aria-label={`${holding.name} 수정`}
                  >
                    수정
                  </button>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

function sign(value: number | null): string {
  if (value === null || value === 0) return ''
  return value > 0 ? 'gain' : 'loss'
}

// 글자색은 증권사 관례대로 전일 종가 대비다(상승 빨강·하락 파랑). 플래시는
// 직전 조회값 대비라 기준이 다르다 — 하락장에서 살짝 오르면 파란 글자에
// 빨간 플래시가 뜨는 게 정상이다.
// 통화 표시(원화/달러) 전환은 quote.price 자체를 바꾸지 않으니, 그걸 기준으로
// 비교해야 통화 토글만으로 애니메이션이 오작동하지 않는다.
function PriceCell({
  quotePrice,
  prevClose,
  formatted,
}: {
  quotePrice: number | null
  prevClose: number | null
  formatted: string
}) {
  const prevRef = useRef<number | null>(null)
  const [flash, setFlash] = useState<'up' | 'down' | null>(null)

  useEffect(() => {
    const prev = prevRef.current
    prevRef.current = quotePrice
    if (quotePrice === null || prev === null || quotePrice === prev) return
    setFlash(quotePrice > prev ? 'up' : 'down')
    const id = setTimeout(() => setFlash(null), 700)
    return () => clearTimeout(id)
  }, [quotePrice])

  const daily =
    quotePrice === null || !prevClose || quotePrice === prevClose
      ? ''
      : quotePrice > prevClose
        ? 'gain'
        : 'loss'

  return (
    <td className={`num price-cell ${daily}${flash ? ` flash-${flash}` : ''}`}>{formatted}</td>
  )
}
