import { useState } from 'react'
import { won } from '../format'
import type { Category, Holding, ManualAccount, ManualHolding } from '../types'
import { CATEGORIES, MANUAL_ACCOUNT_PREFIX, MANUAL_HOLDING_PREFIX } from '../types'

const NO_ACCOUNT = ''

interface Input {
  name: string
  category: Category
  evalAmount: number
  accountId?: string
}

interface Props {
  manualHoldings: ManualHolding[]
  manualAccounts: ManualAccount[]
  holdings: Holding[]
  busy: boolean
  onAdd: (input: Input) => void
  onUpdate: (id: string, input: Input) => void
  onRemove: (id: string) => void
}

// 계좌를 만들지 않고도 종목 하나만 던져 넣을 수 있다. 분류는 여기서 따로 관리하지
// 않는다 — 추가·수정 시 종목명으로 overrides 에 얹어서 보내면, 서버가 다른 종목과
// 똑같은 방식으로 분류해 응답의 holdings 에 섞어 돌려준다. accountId 를 고르면
// "계좌 추가"로 만든 계좌에 붙고, "계좌 없음"이면 이 종목 혼자 자기 몫만 집계된다.
export function ManualAssets({
  manualHoldings,
  manualAccounts,
  holdings,
  busy,
  onAdd,
  onUpdate,
  onRemove,
}: Props) {
  const [editingId, setEditingId] = useState<string | null>(null)
  const [name, setName] = useState('')
  const [category, setCategory] = useState<Category>(CATEGORIES[0])
  const [amount, setAmount] = useState('')
  const [accountId, setAccountId] = useState(NO_ACCOUNT)

  function findHolding(item: ManualHolding): Holding | undefined {
    // 계좌에 붙은 종목은 같은 계좌번호를 공유하니 이름까지 맞춰야 서로 구분된다.
    const accountNumber = item.accountId
      ? MANUAL_ACCOUNT_PREFIX + item.accountId
      : MANUAL_HOLDING_PREFIX + item.id
    return holdings.find((h) => h.accountNumber === accountNumber && h.name === item.name)
  }

  function startEdit(target: ManualHolding) {
    setEditingId(target.id)
    setName(target.name)
    setCategory(findHolding(target)?.category ?? CATEGORIES[0])
    setAmount(String(target.evalAmount))
    setAccountId(target.accountId ?? NO_ACCOUNT)
  }

  function resetForm() {
    setEditingId(null)
    setName('')
    setCategory(CATEGORIES[0])
    setAmount('')
    setAccountId(NO_ACCOUNT)
  }

  function submit(event: React.FormEvent) {
    event.preventDefault()
    const evalAmount = Number(amount)
    if (!name.trim() || !Number.isFinite(evalAmount) || evalAmount <= 0) return

    const input = {
      name: name.trim(),
      category,
      evalAmount,
      accountId: accountId || undefined,
    }
    if (editingId) {
      onUpdate(editingId, input)
    } else {
      onAdd(input)
    }
    resetForm()
  }

  function accountLabel(item: ManualHolding): string | null {
    if (!item.accountId) return null
    return manualAccounts.find((a) => a.id === item.accountId)?.name ?? null
  }

  return (
    <section className="manual">
      <header className="section-head">
        <h2>종목 추가</h2>
      </header>
      <p className="manual-hint">잔고파일에 없는 예금·부동산·코인 등을 종목 하나로 더한다.</p>

      {manualHoldings.length > 0 && (
        <ul className="manual-list">
          {manualHoldings.map((item) => (
            <li key={item.id}>
              <span className="manual-name">
                {item.name}
                {accountLabel(item) && <span className="manual-account-tag"> · {accountLabel(item)}</span>}
              </span>
              <span className="manual-category">{findHolding(item)?.category}</span>
              <span className="manual-amount">{won(item.evalAmount)}</span>
              <span className="manual-actions">
                <button type="button" className="link" disabled={busy} onClick={() => startEdit(item)}>
                  수정
                </button>
                <button
                  type="button"
                  className="link"
                  disabled={busy}
                  onClick={() => {
                    if (editingId === item.id) resetForm()
                    onRemove(item.id)
                  }}
                >
                  삭제
                </button>
              </span>
            </li>
          ))}
        </ul>
      )}

      <form className="manual-form" onSubmit={submit}>
        <input
          value={name}
          onChange={(event) => setName(event.target.value)}
          placeholder="종목명 (예: 정기예금)"
          disabled={busy}
          required
        />
        <select
          value={category}
          onChange={(event) => setCategory(event.target.value as Category)}
          disabled={busy}
          aria-label="분류"
        >
          {CATEGORIES.map((c) => (
            <option key={c} value={c}>
              {c}
            </option>
          ))}
        </select>
        <input
          value={amount}
          onChange={(event) => setAmount(event.target.value)}
          placeholder="평가금액"
          inputMode="numeric"
          disabled={busy}
          required
        />
        <select
          value={accountId}
          onChange={(event) => setAccountId(event.target.value)}
          disabled={busy}
          aria-label="소속 계좌"
        >
          <option value={NO_ACCOUNT}>계좌 없음</option>
          {manualAccounts.map((a) => (
            <option key={a.id} value={a.id}>
              {a.name}
            </option>
          ))}
        </select>
        <button type="submit" disabled={busy}>
          {editingId ? '수정 완료' : '추가'}
        </button>
        {editingId && (
          <button type="button" className="link" disabled={busy} onClick={resetForm}>
            취소
          </button>
        )}
      </form>
    </section>
  )
}
