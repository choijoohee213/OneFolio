import type { Category, Summary } from './types'

export interface Quote {
  price: number
  currency: string
  prevClose?: number
}

/** 파일/수동입력 원본은 그대로 두고, 화면에 보여줄 값만 현재가로 다시 낸다.
 *  저장하지 않으니 토글을 꺼서 언제든 원래 값으로 돌아갈 수 있다. */
export function applyLiveQuotes(
  summary: Summary,
  quotes: Record<string, Quote>,
  usdKrw: number | null,
): Summary {
  let coveredAssetDelta = 0
  const accountDeltas = new Map<string, number>()

  const nextHoldings = summary.holdings.map((h) => {
    if (!h.code) return h
    const quote = quotes[h.code]
    if (!quote) return h
    const fx = quote.currency === 'KRW' ? 1 : usdKrw
    if (!fx) return h

    const evalAmount = quote.price * h.quantity * fx
    const delta = evalAmount - h.evalAmount
    coveredAssetDelta += delta
    accountDeltas.set(h.accountNumber, (accountDeltas.get(h.accountNumber) ?? 0) + delta)

    const buyAmount = h.avgBuyPrice != null ? h.avgBuyPrice * h.quantity : null
    const profitLoss = buyAmount != null ? evalAmount - buyAmount : h.profitLoss
    const profitRate = buyAmount ? (profitLoss! / buyAmount) * 100 : h.profitRate
    const currentPrice = h.quantity !== 0 ? evalAmount / h.quantity : h.currentPrice

    return { ...h, evalAmount, profitLoss, profitRate, currentPrice }
  })

  const coveredAsset = summary.coveredAsset + coveredAssetDelta

  const nextAccounts = summary.accounts.map((a) => {
    const delta = accountDeltas.get(a.number)
    return delta ? { ...a, totalAsset: a.totalAsset + delta } : a
  })

  const weighted = nextHoldings.map((h) => ({
    ...h,
    weight: coveredAsset > 0 ? (h.evalAmount / coveredAsset) * 100 : 0,
  }))

  const amountByCategory = new Map<Category, number>()
  let holdingsTotal = 0
  for (const h of weighted) {
    amountByCategory.set(h.category, (amountByCategory.get(h.category) ?? 0) + h.evalAmount)
    holdingsTotal += h.evalAmount
  }
  const cash = coveredAsset - holdingsTotal
  if (coveredAsset > 0 && cash !== 0) {
    amountByCategory.set('현금성', (amountByCategory.get('현금성') ?? 0) + cash)
  }
  const categories = [...amountByCategory.entries()]
    .map(([category, amount]) => ({
      category,
      amount,
      weight: coveredAsset > 0 ? (amount / coveredAsset) * 100 : 0,
    }))
    .sort((a, b) => b.amount - a.amount)

  return {
    ...summary,
    coveredAsset,
    totalAsset: summary.totalAsset + coveredAssetDelta,
    accounts: nextAccounts,
    categories,
    holdings: [...weighted].sort((a, b) => b.evalAmount - a.evalAmount),
  }
}
