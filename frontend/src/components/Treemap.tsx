import { useMemo, useRef, useState } from 'react'
import { percent, signedPercent, signedWon, won } from '../format'
import { byHolding, type HoldingTotal } from '../charts'
import type { Holding } from '../types'

// 칸을 %로 배치한다. SVG 로 그리면 뷰박스를 늘리는 만큼 글자까지 늘어나
// 가로로 퍼진다 — 칸만 비율로 두고 글자는 일반 텍스트로 얹는다.
const WIDTH = 100
const HEIGHT = 62
// 손익률 색이 이 값에서 가장 진하다. ±수백 %인 종목 하나 때문에 나머지가
// 전부 흐려지는 걸 막으려고 한계를 둔다.
const FULL_RATE = 30
const NAME_MIN_WIDTH = 9
const NAME_MIN_HEIGHT = 6
const RATE_MIN_HEIGHT = 11

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
    <div className="treemap-wrap" ref={wrapper}>
      <div className="treemap" role="list">
        {cells.map((cell) => {
          const weight = weightOf(cell.total, coveredAsset)
          return (
            <button
              key={cell.total.name}
              type="button"
              role="listitem"
              className={`cell ${active === cell.total.name ? 'on' : ''}`}
              style={{
                left: `${cell.x}%`,
                top: `${cell.y}%`,
                width: `${cell.width}%`,
                height: `${cell.height}%`,
                background: rateColor(cell.total.profitRate),
              }}
              aria-label={`${cell.total.name} ${percent(weight)}, ${won(cell.total.evalAmount)}${
                cell.total.profitRate === null ? '' : `, ${signedPercent(cell.total.profitRate)}`
              }`}
              onMouseEnter={(event) => {
                setActive(cell.total.name)
                moveTip(event)
              }}
              onMouseMove={moveTip}
              onMouseLeave={() => setActive(null)}
              onFocus={() => setActive(cell.total.name)}
              onBlur={() => setActive(null)}
            >
              {cell.width >= NAME_MIN_WIDTH && cell.height >= NAME_MIN_HEIGHT && (
                <span className="cell-name">{cell.total.name}</span>
              )}
              {cell.height >= RATE_MIN_HEIGHT && cell.total.profitRate !== null && (
                <span className="cell-rate">{signedPercent(cell.total.profitRate)}</span>
              )}
            </button>
          )
        })}
      </div>

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
  )
}

function weightOf(total: HoldingTotal, coveredAsset: number): number {
  return coveredAsset > 0 ? (total.evalAmount / coveredAsset) * 100 : 0
}

// 손익률이 없는 종목(예수금 등)은 색을 입히지 않는다 — 0%로 칠하면 본전인
// 것처럼 읽힌다. 섞는 건 oklab 이라야 중간 단계에서 채도가 죽지 않는다.
//
// 아래한계가 60%로 높은 건 글자 때문이다. 칸 글자는 흰색으로 고정하는데,
// 라이트 모드에서 옅게 섞으면 배경이 밝아져 흰 글자가 묻힌다.
function rateColor(rate: number | null): string {
  if (rate === null) return 'var(--series-other)'
  const strength = Math.min(Math.abs(rate) / FULL_RATE, 1)
  const mix = 60 + strength * 40
  return `color-mix(in oklab, var(--${rate >= 0 ? 'gain' : 'loss'}) ${mix.toFixed(0)}%, var(--surface))`
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
