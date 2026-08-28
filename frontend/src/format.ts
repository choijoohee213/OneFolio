import type { Category } from './types'

export function won(value: number): string {
  return `${Math.round(value).toLocaleString('ko-KR')}원`
}

export function signedWon(value: number): string {
  return `${value > 0 ? '+' : ''}${won(value)}`
}

export function usd(value: number): string {
  return `$${value.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
}

export function signedUsd(value: number): string {
  return `${value > 0 ? '+' : ''}${usd(value)}`
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

/** 입력 중인 숫자 문자열에 천 단위 콤마를 넣는다. 소수점 이하도 유지한다. */
export function commaFormat(raw: string): string {
  const clean = raw.replace(/[^\d.]/g, '')
  const [integer, ...rest] = clean.split('.')
  const formatted = integer.replace(/\B(?=(\d{3})+(?!\d))/g, ',')
  return rest.length > 0 ? `${formatted}.${rest.join('')}` : formatted
}

/** 콤마가 들어간 문자열을 숫자로 변환한다. */
export function commaParse(formatted: string): number {
  return Number(formatted.replace(/,/g, ''))
}

// 색은 카테고리 정체성에 고정한다. 금액 순위가 바뀌어도 색이 따라 움직이면 안 된다.
// 슬롯 번호는 목록 순서가 아니라 색끼리 얼마나 잘 갈리는지로 정했다 — 목록 순서를
// 그대로 쓰면 파랑과 보라가 나란히 놓여 색각이상에서 구분되지 않는다.
const COLOR_SLOT: Record<Category, number> = {
  '개별주(국내)': 1,
  '테마·섹터 ETF': 2,
  '지수 ETF': 3,
  '레버리지·인버스': 4,
  '개별주(해외)': 5,
  현금성: 6,
  채권: 7,
}

export function categoryColor(category: Category): string {
  return `var(--cat-${COLOR_SLOT[category] ?? 1})`
}
