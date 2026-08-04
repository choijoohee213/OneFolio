import { useRef, useState } from 'react'

interface Props {
  onFiles: (files: File[]) => void
  busy: boolean
  compact: boolean
}

export function FileDrop({ onFiles, busy, compact }: Props) {
  const input = useRef<HTMLInputElement>(null)
  const [dragging, setDragging] = useState(false)

  function handleDrop(event: React.DragEvent) {
    event.preventDefault()
    setDragging(false)
    const files = [...event.dataTransfer.files]
    if (files.length > 0) onFiles(files)
  }

  return (
    <div
      className={`drop ${dragging ? 'dragging' : ''} ${compact ? 'compact' : ''}`}
      onDragOver={(event) => {
        event.preventDefault()
        setDragging(true)
      }}
      onDragLeave={() => setDragging(false)}
      onDrop={handleDrop}
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

      <p className="drop-title">
        {busy ? '계산하는 중…' : '잔고파일을 여기에 끌어다 놓으세요'}
      </p>
      {!compact && (
        <p className="drop-hint">
          계좌마다 따로 받은 파일을 <strong>한 번에 모두</strong> 선택해야 전체 자산이 나옵니다.
        </p>
      )}
      <button type="button" onClick={() => input.current?.click()} disabled={busy}>
        파일 선택
      </button>
    </div>
  )
}
