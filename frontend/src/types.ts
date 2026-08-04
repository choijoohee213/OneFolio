export const CATEGORIES = [
  '개별주(국내)',
  '개별주(해외)',
  '지수 ETF',
  '레버리지·테마 ETF',
  '현금성',
] as const

export type Category = (typeof CATEGORIES)[number]

export interface CategoryTotal {
  category: Category
  amount: number
  weight: number
}

export interface Holding {
  accountNumber: string
  name: string
  category: Category
  quantity: number
  currentPrice: number | null
  avgBuyPrice: number | null
  buyAmount: number | null
  evalAmount: number
  profitLoss: number | null
  profitRate: number | null
  weight: number
}

export interface Summary {
  totalAsset: number
  categories: CategoryTotal[]
  holdings: Holding[]
}

export type Overrides = Record<string, Category>
