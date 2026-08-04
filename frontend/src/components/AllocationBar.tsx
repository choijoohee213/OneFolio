import { categoryColor, percent, won } from '../format'
import type { CategoryTotal } from '../types'
import { CATEGORIES } from '../types'

const LABEL_MIN_WIDTH = 12

interface Props {
  categories: CategoryTotal[]
}

// 세그먼트는 항상 카테고리 고정 순서로 그린다. 금액 순으로 그리면 인접 색 조합이
// 데이터에 따라 바뀌어서 색 구분 검증이 무의미해진다.
export function AllocationBar({ categories }: Props) {
  const byCategory = new Map(categories.map((total) => [total.category, total]))
  const segments = CATEGORIES.map((category) => byCategory.get(category)).filter(
    (total): total is CategoryTotal => total !== undefined && total.weight > 0,
  )

  return (
    <section className="allocation">
      <div className="allocation-bar" role="img" aria-label={ariaLabel(segments)}>
        {segments.map((segment) => (
          <div
            key={segment.category}
            className="segment"
            style={{ width: `${segment.weight}%`, background: categoryColor(segment.category) }}
          >
            {segment.weight >= LABEL_MIN_WIDTH && (
              <span className="segment-label">{percent(segment.weight)}</span>
            )}
          </div>
        ))}
      </div>

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
    </section>
  )
}

function ariaLabel(segments: CategoryTotal[]): string {
  return `자산배분: ${segments.map((s) => `${s.category} ${percent(s.weight)}`).join(', ')}`
}
