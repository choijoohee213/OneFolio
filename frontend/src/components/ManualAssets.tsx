import { useState } from 'react'
import { won } from '../format'
import type { Category, Holding, ManualHolding } from '../types'
import { CATEGORIES, MANUAL_PREFIX } from '../types'

interface Input {
  name: string
  category: Category
  evalAmount: number
}

interface Props {
  manualHoldings: ManualHolding[]
  holdings: Holding[]
  busy: boolean
  onAdd: (input: Input) => void
  onUpdate: (id: string, input: Input) => void
  onRemove: (id: string) => void
}

// 잔고파일에는 없지만 자산배분에 함께 넣고 싶은 것들 — 예금, 부동산, 코인 등을
// 직접 추가한다. 분류는 여기서 따로 관리하지 않는다. 추가·수정 시 종목명으로
// overrides 에 얹어서 보내면, 서버가 다른 종목과 똑같은 방식으로 분류해 응답의
// holdings 에 섞어 돌려준다 — 그래서 종목 표·파이차트에도 자동으로 반영된다.
export function ManualAssets({ manualHoldings, holdings, busy, onAdd, onUpdate, onRemove }: Props) {
  const [editingId, setEditingId] = useState<string | null>(null)
  const [name, setName] = useState('')
  const [category, setCategory] = useState<Category>(CATEGORIES[0])
  const [amount, setAmount] = useState('')

  function categoryOf(id: string): Category | undefined {
    return holdings.find((h) => h.accountNumber === MANUAL_PREFIX + id)?.category
  }

  function startEdit(target: ManualHolding) {
    setEditingId(target.id)
    setName(target.name)
    setCategory(categoryOf(target.id) ?? CATEGORIES[0])
    setAmount(String(target.evalAmount))
  }

  function resetForm() {
    setEditingId(null)
    setName('')
    setCategory(CATEGORIES[0])
    setAmount('')
  }

  function submit(event: React.FormEvent) {
    event.preventDefault()
    const evalAmount = Number(amount)
    if (!name.trim() || !Number.isFinite(evalAmount) || evalAmount <= 0) return

    const input = { name: name.trim(), category, evalAmount }
    if (editingId) {
      onUpdate(editingId, input)
    } else {
      onAdd(input)
    }
    resetForm()
  }

  return (
    <section className="manual">
      <header className="section-head">
        <h2>직접 추가한 자산</h2>
      </header>
      <p className="manual-hint">잔고파일에 없는 예금·부동산·코인 등을 더해 함께 집계합니다.</p>

      {manualHoldings.length > 0 && (
        <ul className="manual-list">
          {manualHoldings.map((item) => (
            <li key={item.id}>
              <span className="manual-name">{item.name}</span>
              <span className="manual-category">{categoryOf(item.id)}</span>
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
