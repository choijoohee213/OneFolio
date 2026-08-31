import type { Holding } from './types'

/** 달러 보기에서 원화 값을 나눌 환율.
 *
 *  캡처에 달러 원가가 적혀 있었으면 그 값이 함축하는 환율을 쓴다. 증권사가 쓴
 *  환율이라 화면에 있던 달러 값이 그대로 되살아난다 — 지금 환율로 나누면 몇 %
 *  어긋난 값이 나온다.
 *
 *  실시간 시세를 켠 동안에는 금액 자체가 지금 시세로 다시 낸 값이므로 지금
 *  환율을 쓴다. */
export function usdRate(
  holding: Holding,
  usdKrw: number | null | undefined,
  showLive: boolean | undefined,
): number | null {
  if (!showLive) {
    const captured = capturedRate(holding)
    if (captured) return captured
  }
  return usdKrw ?? null
}

function capturedRate(holding: Holding): number | null {
  const { usdAvgBuyPrice, usdCurrentPrice, avgBuyPrice, evalAmount, quantity } = holding
  if (usdAvgBuyPrice && avgBuyPrice) return avgBuyPrice / usdAvgBuyPrice
  if (usdCurrentPrice && quantity) return evalAmount / quantity / usdCurrentPrice
  return null
}
