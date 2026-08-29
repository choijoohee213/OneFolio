import type {
  OcrResult,
  HoldingEdit,
  ManualAccount,
  ManualHolding,
  Overrides,
  StockEntry,
  StockMappings,
  Summary,
  UploadedFile,
} from './types'

const BASE = import.meta.env.VITE_API_BASE ?? ''

/** 배포된 백엔드는 한동안 아무도 안 부르면 잠든다. 잠든 뒤 첫 요청은 깨어나기를
 *  기다리느라 한참 걸린다. 화면을 열자마자 한 번 두드려 두면 사용자가 파일을
 *  고르는 동안 깨어난다. 깨우는 게 목적이라 응답도 실패도 보지 않는다. */
export function warmUp(): void {
  void fetch(`${BASE}/health`).catch(() => {})
}

export async function searchStocks(query: string): Promise<StockEntry[]> {
  const response = await fetch(`${BASE}/api/stocks?q=${encodeURIComponent(query)}`)
  if (!response.ok) return []
  return response.json()
}

export async function fetchPortfolio(
  files: UploadedFile[],
  overrides: Overrides,
  manualAccounts: ManualAccount[],
  manualHoldings: ManualHolding[],
  holdingEdits: HoldingEdit[],
  stockMappings?: StockMappings,
): Promise<Summary> {
  const form = new FormData()
  files.forEach((file) => form.append('files', new Blob([file.data]), file.name))
  if (Object.keys(overrides).length > 0) {
    form.append('overrides', JSON.stringify(overrides))
  }
  if (manualAccounts.length > 0) {
    form.append(
      'manualAccounts',
      JSON.stringify(
        manualAccounts.map(({ id, name, totalAsset, accountNumber }) => ({
          id,
          name,
          totalAsset,
          accountNumber: accountNumber ?? '',
        })),
      ),
    )
  }
  if (manualHoldings.length > 0) {
    form.append(
      'manualHoldings',
      JSON.stringify(
        manualHoldings.map((h) => ({
          id: h.id,
          name: h.name,
          evalAmount: h.evalAmount,
          accountId: h.accountId ?? '',
          quantity: h.quantity ?? null,
          avgBuyPrice: h.avgBuyPrice ?? null,
          buyAmount: h.buyAmount ?? null,
          profitLoss: h.profitLoss ?? null,
          profitRate: h.profitRate ?? null,
        })),
      ),
    )
  }

  if (holdingEdits.length > 0) {
    form.append(
      'holdingEdits',
      JSON.stringify(
        holdingEdits.map(({ accountNumber, name, quantity, avgBuyPrice, evalAmount }) => ({
          accountNumber,
          name,
          quantity,
          avgBuyPrice,
          evalAmount,
        })),
      ),
    )
  }

  if (stockMappings && Object.keys(stockMappings).length > 0) {
    form.append('stockMappings', JSON.stringify(stockMappings))
  }

  const response = await fetch(`${BASE}/api/portfolio`, { method: 'POST', body: form })
  if (!response.ok) {
    throw new Error(await readError(response))
  }
  return response.json()
}

export async function toUploadedFiles(files: File[]): Promise<UploadedFile[]> {
  return Promise.all(
    files.map(async (file) => ({ name: file.name, data: await file.arrayBuffer(), accounts: [] })),
  )
}

export async function extractFromScreenshot(image: File): Promise<OcrResult> {
  const form = new FormData()
  form.append('image', image)
  const response = await fetch(`${BASE}/api/ocr`, { method: 'POST', body: form })
  if (!response.ok) {
    throw new Error(await readError(response))
  }
  return await response.json()
}

export interface QuotesResult {
  quotes: Record<string, { price: number; currency: string; prevClose?: number }>
  usdKrw?: number
}

export async function fetchQuotes(codes: string[]): Promise<QuotesResult> {
  const response = await fetch(`${BASE}/api/quotes`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ codes }),
  })
  if (!response.ok) {
    throw new Error(await readError(response))
  }
  return await response.json()
}

// 400·422 는 {"error"} JSON 이지만 404·405 는 라우터 기본 응답이라 평문이다.
async function readError(response: Response): Promise<string> {
  const body = await response.text()
  try {
    const parsed = JSON.parse(body)
    if (typeof parsed?.error === 'string') return parsed.error
  } catch {
    // 평문 응답
  }
  return body.trim() || `요청 실패 (${response.status})`
}
