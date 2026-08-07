import { categoryColor, percent, quantity, signedPercent, signedWon, won } from '../format'
import type { Category, Holding, Overrides } from '../types'
import { CATEGORIES } from '../types'

interface Props {
  holdings: Holding[]
  overrides: Overrides
  busy: boolean
  onCategoryChange: (name: string, category: Category | null) => void
}

export function HoldingRows({ holdings, overrides, busy, onCategoryChange }: Props) {
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
                <CategoryPicker
                  holding={holding}
                  overridden={holding.name in overrides}
                  busy={busy}
                  onChange={onCategoryChange}
                />
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

interface PickerProps {
  holding: Holding
  overridden: boolean
  busy: boolean
  onChange: (name: string, category: Category | null) => void
}

function CategoryPicker({ holding, overridden, busy, onChange }: PickerProps) {
  return (
    <span className={`category ${overridden ? 'overridden' : ''}`}>
      <span className="swatch small" style={{ background: categoryColor(holding.category) }} />
      <select
        value={holding.category}
        disabled={busy}
        aria-label={`${holding.name} 분류`}
        onChange={(event) => onChange(holding.name, event.target.value as Category)}
      >
        {CATEGORIES.map((category) => (
          <option key={category} value={category}>
            {category}
          </option>
        ))}
      </select>
      {overridden && (
        <button
          type="button"
          className="revert"
          disabled={busy}
          title="자동 분류로 되돌리기"
          aria-label={`${holding.name} 자동 분류로 되돌리기`}
          onClick={() => onChange(holding.name, null)}
        >
          ↺
        </button>
      )}
    </span>
  )
}

function sign(value: number | null): string {
  if (value === null || value === 0) return ''
  return value > 0 ? 'gain' : 'loss'
}
