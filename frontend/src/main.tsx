import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'

// index.html 이 깔아 둔 로딩 화면을 최소 이만큼은 띄워 둔다. 이걸 안 하면
// 앱이 로딩 화면보다 빨리 떠서(빠른 연결에서 0.24초) 아무도 못 본다.
// 그만큼 첫 화면이 늦어진다 — 보여 주기로 하고 감수하는 값이다.
const MIN_SPLASH_MS = 900

function start() {
  createRoot(document.getElementById('root')!).render(
    <StrictMode>
      <App />
    </StrictMode>,
  )
}

// performance.now() 는 페이지를 열기 시작한 순간부터 잰 값이라 그대로 쓴다.
const remaining = MIN_SPLASH_MS - performance.now()
if (remaining > 0) setTimeout(start, remaining)
else start()
