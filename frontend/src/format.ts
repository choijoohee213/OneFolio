import type { Category } from './types'
import { CATEGORIES } from './types'

export function won(value: number): string {
  return `${Math.round(value).toLocaleString('ko-KR')}원`
}

// 도넛 가운데처럼 폭이 좁은 자리용. 정확한 금액은 범례와 계좌 목록에 그대로 있다.
export function wonCompact(value: number): string {
  const abs = Math.abs(value)
  if (abs >= 1e8) return `${(value / 1e8).toFixed(2)}억원`
  if (abs >= 1e4) return `${Math.round(value / 1e4).toLocaleString('ko-KR')}만원`
  return won(value)
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
