import { useEffect, useState } from 'react'

// 화면 높이에 견줘 정한다. 고정값을 쓰면 모니터에서는 늘 떠 있고 휴대폰에서는
// 한참 내려가야 뜬다 — 반 화면쯤 내려갔으면 위로 돌아갈 길이 멀다고 본다.
const SHOW_AFTER_RATIO = 0.6

// 종목이 많으면 표가 한참 내려가는데, 탭은 헤더와 달리 붙어 있지 않아 다른 탭으로
// 옮기려면 도로 위까지 올라와야 한다. 좀 내려갔을 때만 뜬다.
export function ScrollTopButton() {
  const [shown, setShown] = useState(false)

  useEffect(() => {
    function onScroll() {
      setShown(window.scrollY > window.innerHeight * SHOW_AFTER_RATIO)
    }
    onScroll()
    window.addEventListener('scroll', onScroll, { passive: true })
    return () => window.removeEventListener('scroll', onScroll)
  }, [])

  return (
    <button
      type="button"
      className={`scroll-top${shown ? ' shown' : ''}`}
      aria-label="맨 위로"
      // behavior:'smooth' 는 브라우저나 OS 가 부드러운 스크롤을 꺼 두면 조용히
      // 아무것도 안 한다 — 버튼이 안 먹는 것처럼 보이느니 바로 올라가는 게 낫다.
      onClick={() => window.scrollTo(0, 0)}
    >
      <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
        <path
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          d="M12 19V6M6 12l6-6 6 6"
        />
      </svg>
    </button>
  )
}
