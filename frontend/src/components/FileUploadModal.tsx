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
    <Modal open={open} title="잔고파일 추가" className="modal-file-upload" onClose={onClose}>
      <p className="modal-desc">
        미래에셋증권 계좌를 이용 중이라면, PC 사이트에서 계좌별 잔고 파일(엑셀)을 내려받아 아래에
        올려주세요.
      </p>
      <p className="modal-desc modal-desc-muted">
        계좌마다 따로 받은 파일을 한 번에 모두 선택해야 전체 자산이 정확히 계산됩니다.
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
