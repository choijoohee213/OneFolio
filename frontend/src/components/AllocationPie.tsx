import { useMemo, useRef, useState } from 'react'
import { percent, won } from '../format'
import { BASES, BASIS_LABEL, toSlices, type Basis, type Slice } from '../charts'
import { Treemap } from './Treemap'
import type { AccountSummary, CategoryTotal, Holding } from '../types'

const SIZE = 220
const CENTER = SIZE / 2
const RADIUS = 96
const GAP_RADIANS = 0.02
const POP_DISTANCE = 7
const LABEL_MIN_WEIGHT = 7

interface Props {
  categories: CategoryTotal[]
  holdings: Holding[]
  accounts: AccountSummary[]
  coveredAsset: number
}

interface Wedge {
  slice: Slice
  start: number
  end: number
  middle: number
}

export function AllocationPie({ categories, holdings, accounts, coveredAsset }: Props) {
  const [basis, setBasis] = useState<Basis>('category')
  const [active, setActive] = useState<string | null>(null)
  const [tip, setTip] = useState({ x: 0, y: 0 })
  const wrapper = useRef<HTMLDivElement>(null)

  // 종목 기준은 트리맵으로 보여준다 — 종목 수가 많으면 파이 조각이 잘게 쪼개져
  // 읽기 어렵고, 트리맵은 같은 자리에서 손익률까지 색으로 얹을 수 있다.
  const asTreemap = basis === 'holding'

  const wedges = useMemo(
    () => (asTreemap ? [] : toWedges(toSlices(basis, categories, holdings, accounts, coveredAsset))),
    [asTreemap, basis, categories, holdings, accounts, coveredAsset],
  )
  const hovered = wedges.find((wedge) => wedge.slice.key === active)

  function moveTip(event: { clientX: number; clientY: number }) {
    const box = wrapper.current?.getBoundingClientRect()
    if (box) setTip({ x: event.clientX - box.left, y: event.clientY - box.top })
  }

  // 키보드로 옮겨왔을 때는 마우스 좌표가 없으니 조각 위치에서 띄운다.
  function tipAtWedge(wedge: Wedge) {
    const box = wrapper.current?.getBoundingClientRect()
    const svg = box ? Math.min(box.width, SIZE) / SIZE : 1
    setTip({
      x: (CENTER + Math.cos(wedge.middle) * RADIUS * 0.6) * svg,
      y: (CENTER + Math.sin(wedge.middle) * RADIUS * 0.6) * svg,
    })
  }

  return (
    <section className="allocation">
      <header className="section-head">
        <h2>자산배분</h2>
        <div className="toggle" role="group" aria-label="차트 기준">
          {BASES.map((option) => (
            <button
              key={option}
              type="button"
              aria-pressed={basis === option}
              onClick={() => {
                setBasis(option)
                setActive(null)
              }}
            >
              {BASIS_LABEL[option]}
            </button>
          ))}
        </div>
      </header>

      {asTreemap ? (
        <>
          <p className="section-note treemap-note">칸 크기는 비중, 색은 손익률입니다</p>
          <Treemap holdings={holdings} coveredAsset={coveredAsset} />
        </>
      ) : (
      <div className="pie-row">
        <div className="pie-wrap" ref={wrapper}>
          <svg
            className="pie"
            viewBox={`0 0 ${SIZE} ${SIZE}`}
            role="img"
            aria-label={`${BASIS_LABEL[basis]}별 자산배분: ${wedges.map((w) => `${w.slice.key} ${percent(w.slice.weight)}`).join(', ')}`}
          >
            <g className="pie-visual">
              {wedges.map((wedge) => {
                const on = wedge.slice.key === active
                return (
                  <path
                    key={wedge.slice.key}
                    className={`slice ${on ? 'on' : ''}`}
                    d={wedgePath(wedge, GAP_RADIANS)}
                    fill={wedge.slice.color}
                    transform={on ? popOut(wedge) : undefined}
                  />
                )
              })}

              {wedges
                .filter((wedge) => wedge.slice.weight >= LABEL_MIN_WEIGHT)
                .map((wedge) => (
                  <text
                    key={wedge.slice.key}
                    className="pie-label"
                    x={CENTER + Math.cos(wedge.middle) * RADIUS * 0.62}
                    y={CENTER + Math.sin(wedge.middle) * RADIUS * 0.62}
                    textAnchor="middle"
                    dominantBaseline="central"
                  >
                    {wedge.slice.weight.toFixed(0)}%
                  </text>
                ))}
            </g>

            {/* 마우스를 받는 층은 따로 둔다. 보이는 조각이 튀어나오면 커서 밑에서
                빠져나가 hover 가 풀렸다 걸렸다 반복하고, 조각 사이 간격도 빈틈이 된다.
                이 층은 틈 없이 원을 덮고 절대 움직이지 않는다. */}
            {wedges.map((wedge) => (
              <path
                key={wedge.slice.key}
                className="hit"
                d={wedgePath(wedge, 0)}
                tabIndex={0}
                role="button"
                aria-label={`${wedge.slice.key} ${percent(wedge.slice.weight)}, ${won(wedge.slice.amount)}`}
                onMouseEnter={(event) => {
                  setActive(wedge.slice.key)
                  moveTip(event)
                }}
                onMouseMove={moveTip}
                onMouseLeave={() => setActive(null)}
                onFocus={() => {
                  setActive(wedge.slice.key)
                  tipAtWedge(wedge)
                }}
                onBlur={() => setActive(null)}
              />
            ))}
          </svg>

          {hovered && (
            <div className="tooltip" style={{ left: tip.x, top: tip.y }} role="status">
              <span className="tooltip-name">
                <span className="swatch small" style={{ background: hovered.slice.color }} />
                {hovered.slice.key}
              </span>
              <span className="tooltip-weight">{percent(hovered.slice.weight)}</span>
              <span className="tooltip-amount">{won(hovered.slice.amount)}</span>
              <span className="tooltip-of">전체 {won(coveredAsset)} 중</span>
            </div>
          )}
        </div>

        <ul className="legend">
          {wedges.map((wedge) => (
            <li
              key={wedge.slice.key}
              className={active && active !== wedge.slice.key ? 'dim' : ''}
              onMouseEnter={() => setActive(wedge.slice.key)}
              onMouseLeave={() => setActive(null)}
            >
              <span className="swatch" style={{ background: wedge.slice.color }} />
              <span className="legend-name">{wedge.slice.key}</span>
              <span className="legend-weight">{percent(wedge.slice.weight)}</span>
              <span className="legend-amount">{won(wedge.slice.amount)}</span>
            </li>
          ))}
        </ul>
      </div>
      )}
    </section>
  )
}

// 12시부터 비중이 큰 순으로 시계방향으로 그린다. 색은 조각 순서가 아니라
// toSlices 가 정해둔 것을 그대로 쓴다 — 분류는 카테고리에, 나머지 기준은
// 금액 순위에 묶인다.
function toWedges(slices: Slice[]): Wedge[] {
  let angle = -Math.PI / 2

  return slices
    .filter((slice) => slice.weight > 0)
    .map((slice) => {
      const span = (slice.weight / 100) * 2 * Math.PI
      const wedge = { slice, start: angle, end: angle + span, middle: angle + span / 2 }
      angle += span
      return wedge
    })
}

function wedgePath({ start, end }: Wedge, gap: number): string {
  // 조각이 아주 얇으면 간격을 빼다가 뒤집힌다.
  const pad = Math.min(gap, (end - start) / 4)
  const from = start + pad
  const to = end - pad
  const large = to - from > Math.PI ? 1 : 0

  const x1 = CENTER + Math.cos(from) * RADIUS
  const y1 = CENTER + Math.sin(from) * RADIUS
  const x2 = CENTER + Math.cos(to) * RADIUS
  const y2 = CENTER + Math.sin(to) * RADIUS

  return `M ${CENTER} ${CENTER} L ${x1} ${y1} A ${RADIUS} ${RADIUS} 0 ${large} 1 ${x2} ${y2} Z`
}

function popOut({ middle }: Wedge): string {
  return `translate(${Math.cos(middle) * POP_DISTANCE} ${Math.sin(middle) * POP_DISTANCE})`
}
