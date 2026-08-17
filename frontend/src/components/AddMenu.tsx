import { useEffect, useRef, useState } from 'react'

interface Props {
  busy: boolean
  onFileUpload: () => void
  onScreenshot: () => void
  onAddAccount: () => void
  onAddHolding: () => void
}

// 잔고파일·스크린샷·계좌·종목 추가를 한곳에 모은 드롭다운. 헤더에 고정으로
// 떠 있어서 페이지 어디를 보고 있든 같은 자리에서 자산을 추가할 수 있다.
export function AddMenu({ busy, onFileUpload, onScreenshot, onAddAccount, onAddHolding }: Props) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    function onDocClick(event: MouseEvent) {
      if (ref.current && !ref.current.contains(event.target as Node)) setOpen(false)
    }
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') setOpen(false)
    }
    document.addEventListener('click', onDocClick)
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('click', onDocClick)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [open])

  function select(action: () => void) {
    setOpen(false)
    action()
  }

  return (
    <div className="add-menu" ref={ref}>
      <button
        type="button"
        className="add-menu-trigger"
        disabled={busy}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label="자산 추가"
        onClick={() => setOpen((v) => !v)}
      >
        +
      </button>
      {open && (
        <ul className="add-menu-list" role="menu">
          <li role="none">
            <button type="button" role="menuitem" onClick={() => select(onFileUpload)}>
              잔고파일 추가
            </button>
          </li>
          <li role="none">
            <button type="button" role="menuitem" onClick={() => select(onScreenshot)}>
              스크린샷으로 추가
            </button>
          </li>
          <li role="none">
            <button type="button" role="menuitem" onClick={() => select(onAddAccount)}>
              계좌 추가
            </button>
          </li>
          <li role="none">
            <button type="button" role="menuitem" onClick={() => select(onAddHolding)}>
              종목 추가
            </button>
          </li>
        </ul>
      )}
    </div>
  )
}
