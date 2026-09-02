import type { Holding } from './types'
import type { Quote } from './liveQuotes'

/** 달러 보기에 쓸 값. 원화 값을 환율로 나누는 대신 달러끼리 계산해 낸다. */
export interface UsdFigures {
  avgBuyPrice: number | null
  currentPrice: number | null
  evalAmount: number
  buyAmount: number | null
  profitLoss: number | null
  profitRate: number | null
}

/** 달러로 보여줄 종목의 값들.
 *
 *  원화 값을 지금 환율로 나누면 두 가지가 어긋난다. 캡처에 달러로 적혀 있던
 *  평단가가 증권사와 몇 % 달라지고, 원화 기준 손익을 나눈 값은 화면의
 *  "평가금액 - 매입금액"과 맞지 않는다(환차손익이 섞여 들어간다).
 *
 *  그래서 달러 값에서 출발해 달러끼리 계산한다. 달러 원가가 없을 때만 환율로
 *  추정한다 — 그때는 원화 값을 나눈 것과 같아진다.
 *
 *  달러로 볼 종목이 아니면 null 이다. */
export function usdFigures(
  holding: Holding,
  quote: Quote | undefined,
  usdKrw: number | null | undefined,
  showLive: boolean | undefined,
): UsdFigures | null {
  if (quote?.currency !== 'USD') return null
  const rate = capturedRate(holding) ?? usdKrw
  if (!rate) return null

  const avgBuyPrice = holding.usdAvgBuyPrice ?? divide(holding.avgBuyPrice, rate)
  // 실시간을 켜면 지금 시세가 곧 현재가다. 꺼져 있으면 캡처에 적혀 있던 값을 쓴다.
  const currentPrice = showLive
    ? quote.price
    : holding.usdCurrentPrice ?? (holding.quantity ? holding.evalAmount / holding.quantity / rate : null)

  const evalAmount =
    currentPrice !== null && holding.quantity ? currentPrice * holding.quantity : holding.evalAmount / rate
  const buyAmount =
    avgBuyPrice !== null && holding.quantity
      ? avgBuyPrice * holding.quantity
      : divide(holding.buyAmount, rate)

  const profitLoss = buyAmount === null ? null : evalAmount - buyAmount
  const profitRate = profitLoss === null || !buyAmount ? null : (profitLoss / buyAmount) * 100

  return { avgBuyPrice, currentPrice, evalAmount, buyAmount, profitLoss, profitRate }
}

function divide(value: number | null | undefined, rate: number): number | null {
  return value === null || value === undefined ? null : value / rate
}

/** 캡처가 함축하는 환율. 증권사가 쓴 환율이라 화면에 있던 달러 값이 되살아난다. */
function capturedRate(holding: Holding): number | null {
  const { usdAvgBuyPrice, usdCurrentPrice, avgBuyPrice, evalAmount, quantity } = holding
  if (usdAvgBuyPrice && avgBuyPrice) return avgBuyPrice / usdAvgBuyPrice
  if (usdCurrentPrice && quantity) return evalAmount / quantity / usdCurrentPrice
  return null
}
