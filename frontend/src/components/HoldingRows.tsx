import { categoryColor, percent, quantity, signedPercent, signedWon, won } from '../format'
import type { Holding } from '../types'

export function HoldingRows({ holdings }: { holdings: Holding[] }) {
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
          </tr>
        </thead>
        <tbody>
          {holdings.map((holding) => (
            <tr key={`${holding.accountNumber}-${holding.name}`}>
              <td className="name">{holding.name}</td>
              <td>
                <span className="swatch small" style={{ background: categoryColor(holding.category) }} />
                {holding.category}
              </td>
              <td className="num">{quantity(holding.quantity)}</td>
              <td className="num">{won(holding.evalAmount)}</td>
              <td className={`num ${sign(holding.profitLoss)}`}>
                {holding.profitLoss === null ? '—' : signedWon(holding.profitLoss)}
              </td>
              <td className={`num ${sign(holding.profitRate)}`}>
                {holding.profitRate === null ? '—' : signedPercent(holding.profitRate)}
              </td>
              <td className="num">{percent(holding.weight)}</td>
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
