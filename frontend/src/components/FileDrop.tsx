import { useRef, useState } from 'react'

interface Props {
  onFiles: (files: File[]) => void
  busy: boolean
}

export function FileDrop({ onFiles, busy }: Props) {
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
      className={`drop-compact ${dragging ? 'dragging' : ''}`}
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
      <p className="drop-title">{busy ? '계산하는 중…' : '잔고파일을 여기에 끌어다 놓으세요'}</p>
      <button type="button" onClick={() => input.current?.click()} disabled={busy}>
        파일 선택
      </button>
    </div>
  )
}
