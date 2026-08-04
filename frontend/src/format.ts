import type { Category } from './types'
import { CATEGORIES } from './types'

export function won(value: number): string {
  return `${Math.round(value).toLocaleString('ko-KR')}원`
}

export function signedWon(value: number): string {
  return `${value > 0 ? '+' : ''}${won(value)}`
}

export function percent(value: number): string {
  return `${value.toFixed(2)}%`
}

export function signedPercent(value: number): string {
  return `${value > 0 ? '+' : ''}${value.toFixed(2)}%`
}

export function quantity(value: number): string {
  return Number.isInteger(value) ? String(value) : value.toLocaleString('ko-KR', { maximumFractionDigits: 4 })
}

// 색은 카테고리 정체성에 고정한다. 금액 순위가 바뀌어도 색이 따라 움직이면 안 된다.
export function categoryColor(category: Category): string {
  return `var(--cat-${CATEGORIES.indexOf(category) + 1})`
}
