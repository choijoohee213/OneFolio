import { useMemo, useRef, useState } from 'react'
import { percent, signedPercent, signedWon, won } from '../format'
import { byHolding, type HoldingTotal } from '../charts'
import type { Holding } from '../types'

const WIDTH = 100
const HEIGHT = 62
const GAP = 0.5
// 손익률 색이 이 값에서 가장 진하다. ±수백 %인 종목 하나 때문에 나머지가
// 전부 흐려지는 걸 막으려고 한계를 둔다.
const FULL_RATE = 30
const NAME_MIN_WIDTH = 13
const NAME_MIN_HEIGHT = 7
const RATE_MIN_HEIGHT = 12

interface Props {
  holdings: Holding[]
  coveredAsset: number
}

interface Cell {
  total: HoldingTotal
  x: number
  y: number
  width: number
  height: number
}

export function Treemap({ holdings, coveredAsset }: Props) {
  const [active, setActive] = useState<string | null>(null)
  const [tip, setTip] = useState({ x: 0, y: 0 })
  const wrapper = useRef<HTMLDivElement>(null)

  const cells = useMemo(() => layout(byHolding(holdings)), [holdings])
  const hovered = cells.find((cell) => cell.total.name === active)

  function moveTip(event: { clientX: number; clientY: number }) {
    const box = wrapper.current?.getBoundingClientRect()
    if (box) setTip({ x: event.clientX - box.left, y: event.clientY - box.top })
  }

  if (cells.length === 0) return null

  return (
    <section className="treemap-section">
      <header className="section-head">
        <h2>종목별 비중과 손익</h2>
        <p className="section-note">칸 크기는 비중, 색은 손익률입니다</p>
      </header>

      <div className="treemap-wrap" ref={wrapper}>
        <svg
          className="treemap"
          viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
          preserveAspectRatio="none"
          role="img"
          aria-label={`종목별 비중과 손익: ${cells
            .map(
              (cell) =>
                `${cell.total.name} ${percent(weightOf(cell.total, coveredAsset))}, ${
                  cell.total.profitRate === null ? '손익 없음' : signedPercent(cell.total.profitRate)
                }`,
            )
            .join(', ')}`}
        >
          {cells.map((cell) => {
            const on = cell.total.name === active
            return (
              <g key={cell.total.name}>
                <rect
                  className={`cell ${on ? 'on' : ''}`}
                  x={cell.x + GAP / 2}
                  y={cell.y + GAP / 2}
                  width={Math.max(0, cell.width - GAP)}
                  height={Math.max(0, cell.height - GAP)}
                  rx={0.6}
                  fill={rateColor(cell.total.profitRate)}
                  tabIndex={0}
                  role="button"
                  aria-label={`${cell.total.name} ${percent(weightOf(cell.total, coveredAsset))}, ${won(cell.total.evalAmount)}`}
                  onMouseEnter={(event) => {
                    setActive(cell.total.name)
                    moveTip(event)
                  }}
                  onMouseMove={moveTip}
                  onMouseLeave={() => setActive(null)}
                  onFocus={() => setActive(cell.total.name)}
                  onBlur={() => setActive(null)}
                />
                {cell.width >= NAME_MIN_WIDTH && cell.height >= NAME_MIN_HEIGHT && (
                  <text className="cell-name" x={cell.x + 1.4} y={cell.y + 3.4}>
                    {clip(cell.total.name, cell.width)}
                  </text>
                )}
                {cell.height >= RATE_MIN_HEIGHT && cell.total.profitRate !== null && (
                  <text className="cell-rate" x={cell.x + 1.4} y={cell.y + 6.6}>
                    {signedPercent(cell.total.profitRate)}
                  </text>
                )}
              </g>
            )
          })}
        </svg>

        {hovered && (
          <div className="tooltip" style={{ left: tip.x, top: tip.y }} role="status">
            <span className="tooltip-name">{hovered.total.name}</span>
            <span className="tooltip-weight">{percent(weightOf(hovered.total, coveredAsset))}</span>
            <span className="tooltip-amount">{won(hovered.total.evalAmount)}</span>
            {hovered.total.profitLoss !== null && (
              <span className={`tooltip-of ${hovered.total.profitLoss > 0 ? 'gain' : 'loss'}`}>
                {signedWon(hovered.total.profitLoss)}
                {hovered.total.profitRate !== null && ` (${signedPercent(hovered.total.profitRate)})`}
              </span>
            )}
          </div>
        )}
      </div>
    </section>
  )
}

function weightOf(total: HoldingTotal, coveredAsset: number): number {
  return coveredAsset > 0 ? (total.evalAmount / coveredAsset) * 100 : 0
}

// 손익률이 없는 종목(현금성 등)은 색을 입히지 않는다 — 0%로 칠하면 본전인
// 것처럼 읽힌다.
function rateColor(rate: number | null): string {
  if (rate === null) return 'var(--series-other)'
  const strength = Math.min(Math.abs(rate) / FULL_RATE, 1)
  const mix = 22 + strength * 78
  return `color-mix(in srgb, var(--${rate >= 0 ? 'gain' : 'loss'}) ${mix.toFixed(0)}%, var(--surface))`
}

// viewBox 단위가 곧 폭이라 글자 수를 폭으로 어림잡는다.
function clip(name: string, width: number): string {
  const max = Math.floor((width - 2.4) / 2.1)
  return name.length > max ? `${name.slice(0, Math.max(1, max - 1))}…` : name
}

// squarified treemap — 칸을 정사각형에 가깝게 만들어 좁고 긴 칸이 생기지 않게 한다.
function layout(totals: HoldingTotal[]): Cell[] {
  const items = totals.filter((total) => total.evalAmount > 0).sort((a, b) => b.evalAmount - a.evalAmount)
  if (items.length === 0) return []

  const sum = items.reduce((total, item) => total + item.evalAmount, 0)
  const areas = items.map((item) => (item.evalAmount / sum) * WIDTH * HEIGHT)

  const cells: Cell[] = []
  let x = 0
  let y = 0
  let width = WIDTH
  let height = HEIGHT
  let index = 0

  while (index < areas.length) {
    const vertical = width >= height
    const side = vertical ? height : width
    const row: number[] = []
    let rowArea = 0

    // 종횡비가 나빠지기 직전까지 한 줄에 채운다.
    while (index + row.length < areas.length) {
      const next = areas[index + row.length]
      if (row.length > 0 && worst(row, rowArea, side) < worst([...row, next], rowArea + next, side)) break
      row.push(next)
      rowArea += next
    }

    const thickness = rowArea / side
    let offset = 0
    for (let i = 0; i < row.length; i++) {
      const length = (row[i] / rowArea) * side
      cells.push({
        total: items[index + i],
        x: vertical ? x : x + offset,
        y: vertical ? y + offset : y,
        width: vertical ? thickness : length,
        height: vertical ? length : thickness,
      })
      offset += length
    }

    if (vertical) {
      x += thickness
      width -= thickness
    } else {
      y += thickness
      height -= thickness
    }
    index += row.length
  }
  return cells
}

// 한 줄에서 가장 찌그러진 칸의 종횡비. 작을수록 정사각형에 가깝다.
function worst(row: number[], rowArea: number, side: number): number {
  const thickness = rowArea / side
  return row.reduce((max, area) => {
    const length = area / thickness
    return Math.max(max, Math.max(length / thickness, thickness / length))
  }, 0)
}
