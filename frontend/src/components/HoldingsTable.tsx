import { categoryColor, percent, quantity, signedPercent, signedWon, won } from '../format'
import type { Holding } from '../types'

export type GroupMode = 'account' | 'name'

interface Props {
  holdings: Holding[]
  mode: GroupMode
  onModeChange: (mode: GroupMode) => void
}

export function HoldingsTable({ holdings, mode, onModeChange }: Props) {
  const rows = mode === 'name' ? mergeByName(holdings) : holdings

  return (
    <section className="holdings">
      <header className="section-head">
        <h2>보유 종목</h2>
        <div className="toggle" role="group" aria-label="종목 묶음 기준">
          <button
            type="button"
            aria-pressed={mode === 'name'}
            onClick={() => onModeChange('name')}
          >
            종목별
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
            {rows.map((holding) => (
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
    </section>
  )
}

function sign(value: number | null): string {
  if (value === null || value === 0) return ''
  return value > 0 ? 'gain' : 'loss'
}

// 같은 종목을 여러 계좌에 나눠 들고 있으면 API 는 계좌마다 별도 행으로 준다.
// 종목별 보기에서는 수량·금액을 합치고, 평단은 합산 매입금액에서 다시 낸다.
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
