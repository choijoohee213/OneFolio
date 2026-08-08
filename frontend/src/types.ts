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

/** 잔고파일 없이 이름과 총액만으로 직접 만든 계좌(예: "저축은행 800만원").
 *  실제 계좌와 똑같이 계산된다 — 종목을 안 붙이면 전액이 현금성으로 잡힌다. */
export interface ManualAccount {
  id: string
  name: string
  totalAsset: number
}

/** 잔고파일에 없는, 사용자가 직접 추가한 종목(예금·부동산·코인 등).
 *  분류는 따로 안 들고 다닌다 — 추가·수정 시 overrides 에 이름으로 얹어서
 *  서버가 다른 종목과 똑같이 분류하고 응답의 holdings 에 섞여 나온다.
 *  accountId 를 주면 직접 만든 계좌에 붙고, 비우면 어느 계좌에도 안 속한 채
 *  자기 몫만 집계에 잡힌다. */
export interface ManualHolding {
  id: string
  name: string
  evalAmount: number
  accountId?: string
}

/** Account.number / Holding.accountNumber 접두사. 직접 추가한 계좌인지,
 *  계좌 없이 던져 넣은 종목인지 구분한다. */
export const MANUAL_ACCOUNT_PREFIX = 'manual-account:'
export const MANUAL_HOLDING_PREFIX = 'manual-item:'

export function isManualAccountNumber(accountNumber: string): boolean {
  return accountNumber.startsWith(MANUAL_ACCOUNT_PREFIX)
}

// 계좌에 붙었든(manual-account:) 안 붙었든(manual-item:) 직접 추가한 종목이면
// 참이다 — 수량 같은 파일 전용 개념이 없다는 걸 표시할 때 쓴다.
export function isManualHolding(holding: Pick<Holding, 'accountNumber'>): boolean {
  return (
    holding.accountNumber.startsWith(MANUAL_HOLDING_PREFIX) ||
    isManualAccountNumber(holding.accountNumber)
  )
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
