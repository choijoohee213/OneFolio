import { categoryColor } from './format'
import type { AccountSummary, CategoryTotal, Holding } from './types'
import { isManualAccountNumber } from './types'

export const BASES = ['category', 'holding', 'account'] as const
export type Basis = (typeof BASES)[number]

export const BASIS_LABEL: Record<Basis, string> = {
  category: '분류',
  holding: '종목',
  account: '계좌',
}

export interface Slice {
  key: string
  amount: number
  weight: number
  color: string
}

// 슬롯 색은 8개까지만 있고 순환하지 않는다. 넘치는 항목은 기타로 접는다.
const SERIES_SLOTS = 8
const OTHER = '기타'

export function toSlices(
  basis: Basis,
  categories: CategoryTotal[],
  holdings: Holding[],
  accounts: AccountSummary[],
  coveredAsset: number,
): Slice[] {
  if (basis === 'category') {
    return categories
      .filter((total) => total.amount > 0)
      .sort((a, b) => b.amount - a.amount)
      .map((total) => ({
        key: total.category,
        amount: total.amount,
        weight: total.weight,
        color: categoryColor(total.category),
      }))
  }

  const amounts = basis === 'holding' ? sumByName(holdings) : sumByAccount(holdings, accounts)
  return withSlotColors(amounts, coveredAsset)
}

function sumByName(holdings: Holding[]): Map<string, number> {
  const amounts = new Map<string, number>()
  for (const holding of holdings) {
    amounts.set(holding.name, (amounts.get(holding.name) ?? 0) + holding.evalAmount)
  }
  return amounts
}

export interface HoldingTotal {
  name: string
  evalAmount: number
  buyAmount: number | null
  profitLoss: number | null
  profitRate: number | null
}

// 같은 종목이 여러 계좌에 있으면 합친다. 손익률은 계좌별 값을 평균 내면 틀리므로
// 합산 매입금액에서 다시 낸다.
export function byHolding(holdings: Holding[]): HoldingTotal[] {
  const merged = new Map<string, HoldingTotal>()

  for (const holding of holdings) {
    const found = merged.get(holding.name)
    if (!found) {
      merged.set(holding.name, {
        name: holding.name,
        evalAmount: holding.evalAmount,
        buyAmount: holding.buyAmount,
        profitLoss: holding.profitLoss,
        profitRate: holding.profitRate,
      })
      continue
    }
    found.evalAmount += holding.evalAmount
    found.buyAmount = addNullable(found.buyAmount, holding.buyAmount)
    found.profitLoss = addNullable(found.profitLoss, holding.profitLoss)
  }

  for (const total of merged.values()) {
    total.profitRate =
      total.buyAmount === null || total.buyAmount === 0 || total.profitLoss === null
        ? null
        : (total.profitLoss / total.buyAmount) * 100
  }
  return [...merged.values()]
}

function addNullable(a: number | null, b: number | null): number | null {
  if (a === null || b === null) return null
  return a + b
}

// 계좌 이름은 계좌번호가 아니라 유형(ISA·연금저축 등)으로 보여준다. 유형이 같은
// 계좌가 여럿이면 한 조각으로 합쳐지는데, 비중을 보는 목적에는 그편이 맞다.
function sumByAccount(holdings: Holding[], accounts: AccountSummary[]): Map<string, number> {
  const nameOf = new Map(accounts.map((account) => [account.number, account.type]))
  const amounts = new Map<string, number>()
  for (const holding of holdings) {
    const name =
      nameOf.get(holding.accountNumber) ??
      (isManualAccountNumber(holding.accountNumber) ? '직접 추가' : '계좌 미지정')
    amounts.set(name, (amounts.get(name) ?? 0) + holding.evalAmount)
  }
  return amounts
}

function withSlotColors(amounts: Map<string, number>, coveredAsset: number): Slice[] {
  const sorted = [...amounts.entries()]
    .filter(([, amount]) => amount > 0)
    .sort((a, b) => b[1] - a[1])

  const shown = sorted.slice(0, SERIES_SLOTS)
  const rest = sorted.slice(SERIES_SLOTS)

  const slices = shown.map(([key, amount], index) => ({
    key,
    amount,
    weight: weightOf(amount, coveredAsset),
    color: `var(--series-${index + 1})`,
  }))

  if (rest.length > 0) {
    const amount = rest.reduce((total, [, value]) => total + value, 0)
    slices.push({
      key: `${OTHER} ${rest.length}개`,
      amount,
      weight: weightOf(amount, coveredAsset),
      color: 'var(--series-other)',
    })
  }

  // 종목으로 잡히지 않은 잔액(예수금)을 채워야 파이가 원을 다 덮는다.
  // 1원 미만은 부동소수점 잔차라 무시한다.
  const held = sorted.reduce((total, [, amount]) => total + amount, 0)
  const deposit = coveredAsset - held
  if (deposit >= 1) {
    slices.push({
      key: '예수금',
      amount: deposit,
      weight: weightOf(deposit, coveredAsset),
      color: 'var(--cat-5)',
    })
  }
  return slices
}

function weightOf(amount: number, coveredAsset: number): number {
  return coveredAsset > 0 ? (amount / coveredAsset) * 100 : 0
}
