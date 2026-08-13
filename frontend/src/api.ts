import type {
  ExtractedHolding,
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
