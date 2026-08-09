import { useState } from 'react'
import { won } from '../format'
import type { AccountSummary, ManualAccount } from '../types'
import { isManualAccountNumber, MANUAL_ACCOUNT_PREFIX } from '../types'

interface AccountInput {
  name: string
  totalAsset: number
  accountNumber?: string
}

interface Props {
  accounts: AccountSummary[]
  manualAccounts: ManualAccount[]
  superseded: ManualAccount[]
  coveredAsset: number
  busy: boolean
  onRemove: (accountNumber: string) => void
  onAddAccount: (input: AccountInput) => void
  onUpdateAccount: (id: string, input: AccountInput) => void
  onRemoveAccount: (id: string) => void
}

export function AccountsPanel({
  accounts,
  manualAccounts,
  superseded,
  coveredAsset,
  busy,
  onRemove,
  onAddAccount,
  onUpdateAccount,
  onRemoveAccount,
}: Props) {
  const [open, setOpen] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [name, setName] = useState('')
  const [amount, setAmount] = useState('')
  const [accountNumber, setAccountNumber] = useState('')

  function startEdit(account: AccountSummary) {
    const id = account.number.slice(MANUAL_ACCOUNT_PREFIX.length)
    setEditingId(id)
    setName(account.type)
    setAmount(String(account.totalAsset))
    setAccountNumber(manualAccounts.find((a) => a.id === id)?.accountNumber ?? '')
    setOpen(true)
  }

  function resetForm() {
    setEditingId(null)
    setName('')
    setAmount('')
    setAccountNumber('')
    setOpen(false)
  }

  function submit(event: React.FormEvent) {
    event.preventDefault()
    const totalAsset = Number(amount)
    if (!name.trim() || !Number.isFinite(totalAsset) || totalAsset <= 0) return

    const input = {
      name: name.trim(),
      totalAsset,
      accountNumber: accountNumber.trim() || undefined,
    }
    if (editingId) {
      onUpdateAccount(editingId, input)
    } else {
      onAddAccount(input)
    }
    resetForm()
  }

  return (
    <section className="accounts">
      <header className="section-head">
        <h2>계좌</h2>
        {!open && (
          <button type="button" className="add-toggle" disabled={busy} onClick={() => setOpen(true)}>
            계좌 추가
          </button>
        )}
      </header>

      {superseded.length > 0 && (
        <p className="notice">
          {superseded.map((a) => a.name).join(', ')} — 같은 계좌번호의 잔고파일이 올라와 파일 쪽으로
          집계했습니다. 직접 적은 총액은 쓰이지 않습니다.
        </p>
      )}

      {accounts.length > 0 && (
        <ul className="account-list">
          {accounts.map((account) => {
            const manual = isManualAccountNumber(account.number)
            return (
              <li key={account.number} className={account.covered ? '' : 'excluded'}>
                <span className="account-type">{account.type}</span>
                <span className={`account-state ${account.covered ? 'on' : ''}`}>
                  {account.covered ? '집계됨' : '종목 없음'}
                </span>
                <span className="account-amount">{won(account.totalAsset)}</span>
                {manual ? (
                  <span className="manual-actions">
                    <button
                      type="button"
                      className="link"
                      disabled={busy}
                      onClick={() => startEdit(account)}
                    >
                      수정
                    </button>
                    <button
                      type="button"
                      className="link"
                      disabled={busy}
                      onClick={() => {
                        const id = account.number.slice(MANUAL_ACCOUNT_PREFIX.length)
                        if (editingId === id) resetForm()
                        onRemoveAccount(id)
                      }}
                    >
                      삭제
                    </button>
                  </span>
                ) : account.covered ? (
                  <button
                    type="button"
                    className="link remove"
                    disabled={busy}
                    onClick={() => onRemove(account.number)}
                    aria-label={`${account.type} 계좌 집계에서 빼기`}
                  >
                    빼기
                  </button>
                ) : (
                  <span className="remove-placeholder" />
                )}
              </li>
            )
          })}
          <li className="account-sum">
            <span className="account-type">집계 기준</span>
            <span className="account-state" />
            <span className="account-amount">{won(coveredAsset)}</span>
            <span className="remove-placeholder" />
          </li>
        </ul>
      )}

      {open && (
        <form className="manual-form" onSubmit={submit}>
          <input
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder="계좌 이름 (예: 저축은행)"
            disabled={busy}
            autoFocus
            required
          />
          <input
            value={accountNumber}
            onChange={(event) => setAccountNumber(event.target.value)}
            placeholder="계좌번호 (선택)"
            disabled={busy}
          />
          <input
            value={amount}
            onChange={(event) => setAmount(event.target.value)}
            placeholder="총자산"
            inputMode="numeric"
            disabled={busy}
            required
          />
          <button type="submit" disabled={busy}>
            {editingId ? '수정 완료' : '추가'}
          </button>
          <button type="button" className="link" disabled={busy} onClick={resetForm}>
            취소
          </button>
        </form>
      )}

      {open && !editingId && (
        <p className="manual-hint">
          계좌번호를 적어 두면 나중에 같은 계좌의 잔고파일을 올렸을 때 중복으로 잡히지 않습니다.
        </p>
      )}
    </section>
  )
}
