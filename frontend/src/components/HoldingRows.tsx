import { categoryColor, percent, quantity, signedPercent, signedWon, won } from '../format'
import type { Holding } from '../types'
import { isManualHolding } from '../types'

interface Props {
  holdings: Holding[]
  busy: boolean
  onEdit: (holding: Holding) => void
}

export function HoldingRows({ holdings, busy, onEdit }: Props) {
  return (
    <div className="scroll">
      <table>
        <thead>
          <tr>
            <th>종목</th>
            <th>분류</th>
            <th className="num">수량</th>
            <th className="num">평가금액</th>
            <th className="num">평가손익</th>
            <th className="num">손익률</th>
            <th className="num">비중</th>
            <th />
          </tr>
        </thead>
        <tbody>
          {holdings.map((holding) => (
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
              <td className="num">{isManualHolding(holding) ? '—' : quantity(holding.quantity)}</td>
              <td className="num">{won(holding.evalAmount)}</td>
              <td className={`num ${sign(holding.profitLoss)}`}>
                {holding.profitLoss === null ? '—' : signedWon(holding.profitLoss)}
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
          ))}
        </tbody>
      </table>
    </div>
  )
}

function sign(value: number | null): string {
  if (value === null || value === 0) return ''
  return value > 0 ? 'gain' : 'loss'
}
