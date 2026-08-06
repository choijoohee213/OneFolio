import { useRef, useState } from 'react'
import { categoryColor, percent, won } from '../format'
import type { Category, CategoryTotal } from '../types'
import { CATEGORIES } from '../types'

const SIZE = 220
const CENTER = SIZE / 2
const RADIUS = 96
const GAP_RADIANS = 0.02
const POP_DISTANCE = 7
const LABEL_MIN_WEIGHT = 7

interface Props {
  categories: CategoryTotal[]
  coveredAsset: number
}

interface Slice {
  total: CategoryTotal
  start: number
  end: number
  middle: number
}

export function AllocationPie({ categories, coveredAsset }: Props) {
  const [active, setActive] = useState<Category | null>(null)
  const [tip, setTip] = useState({ x: 0, y: 0 })
  const wrapper = useRef<HTMLDivElement>(null)

  const slices = toSlices(categories)
  const hovered = slices.find((slice) => slice.total.category === active)

  function moveTip(event: { clientX: number; clientY: number }) {
    const box = wrapper.current?.getBoundingClientRect()
    if (box) setTip({ x: event.clientX - box.left, y: event.clientY - box.top })
  }

  // 키보드로 옮겨왔을 때는 마우스 좌표가 없으니 조각 위치에서 띄운다.
  function tipAtSlice(slice: Slice) {
    const box = wrapper.current?.getBoundingClientRect()
    const svg = box ? Math.min(box.width, SIZE) / SIZE : 1
    setTip({
      x: (CENTER + Math.cos(slice.middle) * RADIUS * 0.6) * svg,
      y: (CENTER + Math.sin(slice.middle) * RADIUS * 0.6) * svg,
    })
  }

  return (
    <section className="allocation">
      <div className="pie-row">
        <div className="pie-wrap" ref={wrapper}>
          <svg
            className="pie"
            viewBox={`0 0 ${SIZE} ${SIZE}`}
            role="img"
            aria-label={`자산배분: ${slices.map((s) => `${s.total.category} ${percent(s.total.weight)}`).join(', ')}`}
          >
            <g className="pie-visual">
              {slices.map((slice) => {
                const on = slice.total.category === active
                return (
                  <path
                    key={slice.total.category}
                    className={`slice ${on ? 'on' : ''}`}
                    d={wedge(slice, GAP_RADIANS)}
                    fill={categoryColor(slice.total.category)}
                    transform={on ? popOut(slice) : undefined}
                  />
                )
              })}

              {slices
                .filter((slice) => slice.total.weight >= LABEL_MIN_WEIGHT)
                .map((slice) => (
                  <text
                    key={slice.total.category}
                    className="pie-label"
                    x={CENTER + Math.cos(slice.middle) * RADIUS * 0.62}
                    y={CENTER + Math.sin(slice.middle) * RADIUS * 0.62}
                    textAnchor="middle"
                    dominantBaseline="central"
                  >
                    {slice.total.weight.toFixed(0)}%
                  </text>
                ))}
            </g>

            {/* 마우스를 받는 층은 따로 둔다. 보이는 조각이 튀어나오면 커서 밑에서
                빠져나가 hover 가 풀렸다 걸렸다 반복하고, 조각 사이 간격도 빈틈이 된다.
                이 층은 틈 없이 원을 덮고 절대 움직이지 않는다. */}
            {slices.map((slice) => (
              <path
                key={slice.total.category}
                className="hit"
                d={wedge(slice, 0)}
                tabIndex={0}
                role="button"
                aria-label={`${slice.total.category} ${percent(slice.total.weight)}, ${won(slice.total.amount)}`}
                onMouseEnter={(event) => {
                  setActive(slice.total.category)
                  moveTip(event)
                }}
                onMouseMove={moveTip}
                onMouseLeave={() => setActive(null)}
                onFocus={() => {
                  setActive(slice.total.category)
                  tipAtSlice(slice)
                }}
                onBlur={() => setActive(null)}
              />
            ))}
          </svg>

          {hovered && (
            <div className="tooltip" style={{ left: tip.x, top: tip.y }} role="status">
              <span className="tooltip-name">
                <span
                  className="swatch small"
                  style={{ background: categoryColor(hovered.total.category) }}
                />
                {hovered.total.category}
              </span>
              <span className="tooltip-weight">{percent(hovered.total.weight)}</span>
              <span className="tooltip-amount">{won(hovered.total.amount)}</span>
              <span className="tooltip-of">전체 {won(coveredAsset)} 중</span>
            </div>
          )}
        </div>

        <ul className="legend">
          {slices.map((slice) => (
            <li
              key={slice.total.category}
              className={active && active !== slice.total.category ? 'dim' : ''}
              onMouseEnter={() => setActive(slice.total.category)}
              onMouseLeave={() => setActive(null)}
            >
              <span
                className="swatch"
                style={{ background: categoryColor(slice.total.category) }}
              />
              <span className="legend-name">{slice.total.category}</span>
              <span className="legend-weight">{percent(slice.total.weight)}</span>
              <span className="legend-amount">{won(slice.total.amount)}</span>
            </li>
          ))}
        </ul>
      </div>
    </section>
  )
}

// 조각은 항상 카테고리 고정 순서다. 금액 순으로 그리면 데이터가 바뀔 때마다
// 맞닿는 색 조합이 달라진다.
function toSlices(categories: CategoryTotal[]): Slice[] {
  const byCategory = new Map(categories.map((total) => [total.category, total]))
  let angle = -Math.PI / 2

  return CATEGORIES.map((category) => byCategory.get(category))
    .filter((total): total is CategoryTotal => total !== undefined && total.weight > 0)
    .map((total) => {
      const span = (total.weight / 100) * 2 * Math.PI
      const slice = { total, start: angle, end: angle + span, middle: angle + span / 2 }
      angle += span
      return slice
    })
}

function wedge({ start, end }: Slice, gap: number): string {
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

function popOut({ middle }: Slice): string {
  return `translate(${Math.cos(middle) * POP_DISTANCE} ${Math.sin(middle) * POP_DISTANCE})`
}
