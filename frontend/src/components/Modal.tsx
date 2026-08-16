import { useEffect, useRef } from 'react'

interface Props {
  open: boolean
  title: string
  wide?: boolean
  onClose: () => void
  children: React.ReactNode
}

// 네이티브 <dialog> 를 쓴다. ESC 닫기, 배경 딤, 포커스 가둠을 브라우저가 해 준다.
export function Modal({ open, title, wide, onClose, children }: Props) {
  const dialog = useRef<HTMLDialogElement>(null)

  useEffect(() => {
    const el = dialog.current
    if (!el) return
    if (open && !el.open) el.showModal()
    if (!open && el.open) el.close()
  }, [open])

  return (
    <dialog
      ref={dialog}
      className={`modal${wide ? ' modal-wide' : ''}`}
      // ESC 나 배경 클릭으로 닫혀도 바깥 상태를 맞춰 둬야 다시 열린다.
      onClose={onClose}
      onClick={(event) => {
        if (event.target === dialog.current) onClose()
      }}
    >
      <div className="modal-body">
        <header className="modal-head">
          <h3>{title}</h3>
          <button type="button" className="modal-close" onClick={onClose} aria-label="닫기">
            ✕
          </button>
        </header>
        {children}
      </div>
    </dialog>
  )
}
