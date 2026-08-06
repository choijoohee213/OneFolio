import { categoryColor, percent, won, wonCompact } from '../format'
import type { CategoryTotal } from '../types'
import { CATEGORIES } from '../types'

const SIZE = 200
const CENTER = SIZE / 2
const RADIUS = 72
const THICKNESS = 34
const CIRCUMFERENCE = 2 * Math.PI * RADIUS
const GAP = 3
const LABEL_MIN_WEIGHT = 9

interface Props {
  categories: CategoryTotal[]
  coveredAsset: number
}

// 조각은 항상 카테고리 고정 순서로 그린다. 금액 순으로 그리면 데이터가 바뀔 때마다
// 맞닿는 색 조합이 달라진다.
export function AllocationDonut({ categories, coveredAsset }: Props) {
  const byCategory = new Map(categories.map((total) => [total.category, total]))
  const segments = CATEGORIES.map((category) => byCategory.get(category)).filter(
    (total): total is CategoryTotal => total !== undefined && total.weight > 0,
  )

  let start = 0
  const arcs = segments.map((segment) => {
    const length = (segment.weight / 100) * CIRCUMFERENCE
    const arc = { segment, length, offset: start, middle: start + length / 2 }
    start += length
    return arc
  })

  return (
    <section className="allocation">
      <div className="donut-row">
        <svg
          className="donut"
          viewBox={`0 0 ${SIZE} ${SIZE}`}
          role="img"
          aria-label={`자산배분: ${segments.map((s) => `${s.category} ${percent(s.weight)}`).join(', ')}`}
        >
          <g transform={`rotate(-90 ${CENTER} ${CENTER})`}>
            {arcs.map(({ segment, length, offset }) => (
              <circle
                key={segment.category}
                cx={CENTER}
                cy={CENTER}
                r={RADIUS}
                fill="none"
                stroke={categoryColor(segment.category)}
                strokeWidth={THICKNESS}
                strokeDasharray={`${Math.max(length - GAP, 0.5)} ${CIRCUMFERENCE}`}
                strokeDashoffset={-offset}
              />
            ))}
          </g>

          {arcs
            .filter(({ segment }) => segment.weight >= LABEL_MIN_WEIGHT)
            .map(({ segment, middle }) => {
              const angle = (middle / CIRCUMFERENCE) * 2 * Math.PI - Math.PI / 2
              return (
                <text
                  key={segment.category}
                  className="donut-label"
                  x={CENTER + Math.cos(angle) * RADIUS}
                  y={CENTER + Math.sin(angle) * RADIUS}
                  textAnchor="middle"
                  dominantBaseline="central"
                >
                  {segment.weight.toFixed(0)}%
                </text>
              )
            })}

          <text className="donut-center-label" x={CENTER} y={CENTER - 10} textAnchor="middle">
            집계 기준
          </text>
          <text className="donut-center-value" x={CENTER} y={CENTER + 12} textAnchor="middle">
            {wonCompact(coveredAsset)}
          </text>
        </svg>

        <ul className="legend">
          {segments.map((segment) => (
            <li key={segment.category}>
              <span className="swatch" style={{ background: categoryColor(segment.category) }} />
              <span className="legend-name">{segment.category}</span>
              <span className="legend-weight">{percent(segment.weight)}</span>
              <span className="legend-amount">{won(segment.amount)}</span>
            </li>
          ))}
        </ul>
      </div>
    </section>
  )
}
