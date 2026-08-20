import { categoryColor, percent, quantity, signedPercent, signedUsd, signedWon, usd, won } from '../format'
import type { Holding } from '../types'
import { isManualHolding } from '../types'
import type { Quote } from '../liveQuotes'

interface Props {
  holdings: Holding[]
  busy: boolean
  onEdit: (holding: Holding) => void
  showUSD?: boolean
  showLive?: boolean
  quotes?: Record<string, Quote> | null
  usdKrw?: number | null
}

export function HoldingRows({ holdings, busy, onEdit, showUSD, showLive, quotes, usdKrw }: Props) {
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
            const fx = showUSD && quote?.currency === 'USD' ? usdKrw ?? null : null
            const amount = (v: number) => (fx ? usd(v / fx) : won(v))
            const signedAmount = (v: number) => (fx ? signedUsd(v / fx) : signedWon(v))
            const currentPrice = () => {
              if (!quote) return '—'
              if (quote.currency === 'USD') return fx ? usd(quote.price) : won(quote.price * (usdKrw ?? 0))
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
                <td className="num">{holding.avgBuyPrice === null ? '—' : amount(holding.avgBuyPrice)}</td>
                {showLive && <td className="num">{currentPrice()}</td>}
                <td className="num">{amount(holding.evalAmount)}</td>
                <td className={`num ${sign(holding.profitLoss)}`}>
                  {holding.profitLoss === null ? '—' : signedAmount(holding.profitLoss)}
                </td>
                <td className={`num ${sign(holding.profitRate)}`}>
                  {holding.profitRate === null ? '—' : signedPercent(holding.profitRate)}
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
