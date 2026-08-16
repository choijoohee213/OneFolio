import { useRef, useState } from 'react'

interface Props {
  onFiles: (files: File[]) => void
  busy: boolean
  variant: 'compact' | 'card'
}

export function FileDrop({ onFiles, busy, variant }: Props) {
  const input = useRef<HTMLInputElement>(null)
  const [dragging, setDragging] = useState(false)

  function handleDrop(event: React.DragEvent) {
    event.preventDefault()
    setDragging(false)
    const files = [...event.dataTransfer.files]
    if (files.length > 0) onFiles(files)
  }

  function openPicker() {
    if (!busy) input.current?.click()
  }

  return (
    <div
      className={`drop drop-${variant} ${variant === 'card' ? 'home-card' : ''} ${dragging ? 'dragging' : ''}`}
      onDragOver={(event) => {
        event.preventDefault()
        setDragging(true)
      }}
      onDragLeave={() => setDragging(false)}
      onDrop={handleDrop}
      onClick={variant === 'card' ? openPicker : undefined}
      onKeyDown={
        variant === 'card'
          ? (event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault()
                openPicker()
              }
            }
          : undefined
      }
      role={variant === 'card' ? 'button' : undefined}
      tabIndex={variant === 'card' ? 0 : undefined}
    >
      <input
        ref={input}
        type="file"
        multiple
        accept=".xls,.xlsx"
        hidden
        onChange={(event) => {
          const files = [...(event.target.files ?? [])]
          if (files.length > 0) onFiles(files)
          event.target.value = ''
        }}
      />

      {variant === 'compact' && (
        <>
          <p className="drop-title">{busy ? '계산하는 중…' : '잔고파일을 여기에 끌어다 놓으세요'}</p>
          <button type="button" onClick={openPicker} disabled={busy}>
            파일 선택
          </button>
        </>
      )}

      {variant === 'card' && (
        <>
          <span className="home-card-title">{busy ? '계산하는 중…' : '잔고파일 추가'}</span>
          <span className="home-card-desc">끌어다 놓거나 클릭해서 선택</span>
        </>
      )}
    </div>
  )
}
