import { useMemo } from 'react'
import { signedPercent, signedWon } from '../format'
import { byHolding } from '../charts'
import type { Holding } from '../types'

interface Props {
  holdings: Holding[]
}

interface Bar {
  name: string
  profitLoss: number
  profitRate: number | null
}

export function ProfitBars({ holdings }: Props) {
  const bars = useMemo(() => toBars(holdings), [holdings])
  if (bars.length === 0) return null

  const total = bars.reduce((sum, bar) => sum + bar.profitLoss, 0)

  // 0 선은 한가운데가 아니라 수익·손실 최대치의 비율로 놓는다. 손실이 하나뿐일 때
  // 왼쪽 절반이 통째로 비는 걸 막는다.
  const maxGain = Math.max(0, ...bars.map((bar) => bar.profitLoss))
  const maxLoss = Math.max(0, ...bars.map((bar) => -bar.profitLoss))
  const span = maxGain + maxLoss
  const zero = span > 0 ? (maxLoss / span) * 100 : 0

  return (
    <section className="profit-bars-section">
      <header className="section-head">
        <h2>손익 기여도</h2>
        <p className={`section-note ${total > 0 ? 'gain' : total < 0 ? 'loss' : ''}`}>
          합계 {signedWon(total)}
        </p>
      </header>

      <ul className="profit-bars">
        {bars.map((bar) => {
          const gain = bar.profitLoss > 0
          const length = span > 0 ? (Math.abs(bar.profitLoss) / span) * 100 : 0
          return (
            <li key={bar.name}>
              <span className="bar-name" title={bar.name}>
                {bar.name}
              </span>
              <span className="bar-track">
                {/* 0 을 가운데 두고 수익은 오른쪽, 손실은 왼쪽으로 뻗는다 */}
                <span
                  className={`bar-fill ${gain ? 'gain' : 'loss'}`}
                  style={{ left: gain ? `${zero}%` : `${zero - length}%`, width: `${length}%` }}
                />
                <span className="bar-zero" style={{ left: `${zero}%` }} aria-hidden="true" />
              </span>
              <span className={`bar-value ${gain ? 'gain' : 'loss'}`}>
                {signedWon(bar.profitLoss)}
                {bar.profitRate !== null && (
                  <span className="bar-rate">{signedPercent(bar.profitRate)}</span>
                )}
              </span>
            </li>
          )
        })}
      </ul>
    </section>
  )
}

// 손익이 큰 순으로 세운다 — 수익이 위, 손실이 아래로 모인다.
function toBars(holdings: Holding[]): Bar[] {
  return byHolding(holdings)
    .filter((total): total is typeof total & { profitLoss: number } => total.profitLoss !== null)
    .filter((total) => total.profitLoss !== 0)
    .map((total) => ({ name: total.name, profitLoss: total.profitLoss, profitRate: total.profitRate }))
    .sort((a, b) => b.profitLoss - a.profitLoss)
}
