import { useState } from 'react'
import { commaFormat, commaParse } from '../format'
import { Modal } from './Modal'
import type { ManualAccount } from '../types'

export interface AccountInput {
  name: string
  totalAsset: number
  accountNumber?: string
}

interface Props {
  open: boolean
  editing: ManualAccount | null
  busy: boolean
  onClose: () => void
  onSubmit: (input: AccountInput) => void
}

export function AccountForm({ open, editing, busy, onClose, onSubmit }: Props) {
  const [name, setName] = useState('')
  const [amount, setAmount] = useState('')
  const [accountNumber, setAccountNumber] = useState('')
  // 모달이 열릴 때마다 대상에 맞춰 채운다. key 로 다시 마운트시키는 대신
  // 열림 전환을 직접 감지해 초기화한다.
  const [wasOpen, setWasOpen] = useState(false)

  if (open !== wasOpen) {
    setWasOpen(open)
    if (open) {
      setName(editing?.name ?? '')
      setAmount(editing ? commaFormat(String(editing.totalAsset)) : '')
      setAccountNumber(editing?.accountNumber ?? '')
    }
  }

  function submit(event: React.FormEvent) {
    event.preventDefault()
    const totalAsset = commaParse(amount)
    if (!name.trim() || !Number.isFinite(totalAsset) || totalAsset <= 0) return
    onSubmit({
      name: name.trim(),
      totalAsset,
      accountNumber: accountNumber.trim() || undefined,
    })
  }

  return (
    <Modal open={open} title={editing ? '계좌 수정' : '계좌 추가'} onClose={onClose}>
      <form className="modal-form" onSubmit={submit}>
        <label>
          <span>계좌 이름</span>
          <input
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder="예: 저축은행"
            disabled={busy}
            autoFocus
            required
          />
        </label>
        <label>
          <span>계좌번호 (선택)</span>
          <input
            value={accountNumber}
            onChange={(event) => setAccountNumber(event.target.value)}
            placeholder="예: 111-1111-1111-0"
            disabled={busy}
          />
          <small>적어 두면 나중에 같은 계좌의 잔고파일을 올렸을 때 중복으로 잡히지 않습니다.</small>
        </label>
        <label>
          <span>총자산</span>
          <input
            value={amount}
            onChange={(event) => setAmount(commaFormat(event.target.value))}
            placeholder="8,000,000"
            inputMode="numeric"
            disabled={busy}
            required
          />
        </label>
        <footer className="modal-actions">
          <button type="button" className="modal-cancel" disabled={busy} onClick={onClose}>
            취소
          </button>
          <button type="submit" disabled={busy}>
            {editing ? '수정 완료' : '추가'}
          </button>
        </footer>
      </form>
    </Modal>
  )
}
