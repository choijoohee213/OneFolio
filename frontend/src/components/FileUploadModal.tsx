import { useRef, useState } from 'react'
import { Modal } from './Modal'

interface Props {
  open: boolean
  busy: boolean
  onClose: () => void
  onFiles: (files: File[]) => void
}

export function FileUploadModal({ open, busy, onClose, onFiles }: Props) {
  const input = useRef<HTMLInputElement>(null)
  const [dragging, setDragging] = useState(false)

  function handleFiles(files: File[]) {
    if (files.length === 0) return
    onFiles(files)
    onClose()
  }

  return (
    <Modal open={open} title="잔고파일 추가" onClose={onClose}>
      <p className="modal-desc">
        해당 잔고파일 추가 기능은 미래에셋증권 사용자에게만 해당되며, 미래에셋증권(PC)에서 계좌별 잔고
        파일 엑셀을 다운받아 파일을 넣어주세요.
      </p>
      <p className="modal-desc modal-desc-muted">
        계좌마다 따로 받은 파일을 한 번에 모두 선택해야 전체 자산이 나옵니다.
      </p>
      <div
        className={`modal-dropzone ${dragging ? 'dragging' : ''}`}
        onDragOver={(event) => {
          event.preventDefault()
          setDragging(true)
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={(event) => {
          event.preventDefault()
          setDragging(false)
          handleFiles([...event.dataTransfer.files])
        }}
        onClick={() => input.current?.click()}
        role="button"
        tabIndex={0}
        onKeyDown={(event) => {
          if (event.key === 'Enter' || event.key === ' ') {
            event.preventDefault()
            input.current?.click()
          }
        }}
      >
        <input
          ref={input}
          type="file"
          multiple
          accept=".xls,.xlsx"
          hidden
          onChange={(event) => {
            handleFiles([...(event.target.files ?? [])])
            event.target.value = ''
          }}
        />
        <p>{busy ? '계산하는 중…' : '잔고파일을 여기에 끌어다 놓거나 클릭해서 선택'}</p>
      </div>
    </Modal>
  )
}
