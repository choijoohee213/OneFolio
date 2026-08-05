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

export interface AccountSummary {
  number: string
  type: string
  totalAsset: number
  /** 이 계좌의 종목 상세가 업로드에 포함됐는지. 잔고파일은 어느 것을 받아도
   *  전체 계좌 요약을 담고 있어서 계좌 목록만으로는 알 수 없다. */
  covered: boolean
}

export interface Source {
  fileName: string
  accountNumbers: string[]
}

/** 브라우저에 쌓아두는 잔고파일. 서버가 상태를 갖지 않아 매번 전부 다시 보낸다. */
export interface UploadedFile {
  name: string
  data: ArrayBuffer
  /** 이 파일이 종목 상세를 담고 있는 계좌. 응답의 sources 로 채운다 */
  accounts: string[]
}

export interface Summary {
  /** 종목 상세가 올라온 계좌들의 자산총액 합. 모든 비중의 분모다 */
  coveredAsset: number
  /** 파일에 적힌 전체 계좌 자산총액 합 */
  totalAsset: number
  accounts: AccountSummary[]
  categories: CategoryTotal[]
  holdings: Holding[]
  /** 보낸 파일 순서대로, 각 파일이 담당한 계좌 */
  sources: Source[]
}

export type Overrides = Record<string, Category>
