import { fetchPortfolio } from './api'
import type { ManualHolding, Overrides, Summary, UploadedFile } from './types'

export interface Collection {
  files: UploadedFile[]
  summary: Summary | null
}

// 서버는 상태를 갖지 않으므로 계좌를 하나 더 올린다고 이전 결과에 더해지지 않는다.
// 올린 파일과 직접 추가한 자산을 브라우저에 쌓아두고 매번 전부 다시 보내야 누적이 된다.
export async function recompute(
  files: UploadedFile[],
  overrides: Overrides,
  manualHoldings: ManualHolding[],
): Promise<Collection> {
  if (files.length === 0 && manualHoldings.length === 0) {
    return { files: [], summary: null }
  }

  const summary = await fetchPortfolio(files, overrides, manualHoldings)
  if (files.length === 0) {
    return { files: [], summary }
  }

  const labelled = files.map((file, index) => ({
    ...file,
    accounts: summary.sources[index]?.accountNumbers ?? [],
  }))

  const kept = keepNewestPerAccount(labelled)
  if (kept.length === labelled.length) {
    return { files: kept, summary }
  }

  // 같은 계좌를 다시 올린 경우다. 낡은 파일에만 있던 종목(그새 판 종목 등)이
  // 남지 않도록 추려낸 파일로 다시 계산한다.
  return { files: kept, summary: await fetchPortfolio(kept, overrides, manualHoldings) }
}

export function withoutAccount(files: UploadedFile[], accountNumber: string): UploadedFile[] {
  return files.filter((file) => !file.accounts.includes(accountNumber))
}

function keepNewestPerAccount(files: UploadedFile[]): UploadedFile[] {
  const claimed = new Set<string>()
  const kept: UploadedFile[] = []

  for (let index = files.length - 1; index >= 0; index--) {
    const file = files[index]
    if (file.accounts.length > 0 && file.accounts.every((account) => claimed.has(account))) {
      continue
    }
    file.accounts.forEach((account) => claimed.add(account))
    kept.unshift(file)
  }

  return kept
}
