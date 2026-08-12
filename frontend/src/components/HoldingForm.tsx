import { useState } from 'react'
import { commaFormat, commaParse } from '../format'
import { Modal } from './Modal'
import { StockSearch } from './StockSearch'
import type { AccountSummary, Category, Holding, ManualAccount, ManualHolding } from '../types'
import { CATEGORIES, isManualAccountNumber } from '../types'

const NO_ACCOUNT = ''

/** 무엇을 편집하는지에 따라 고칠 수 있는 범위가 다르다. */
export type HoldingTarget =
  | { kind: 'new' }
  | { kind: 'manual'; item: ManualHolding; holding?: Holding }
  | { kind: 'file'; holding: Holding }

export interface ManualInput {
  name: string
  category: Category
  evalAmount: number
  accountId?: string
}

export interface FileInput {
  category: Category
  quantity: number
  avgBuyPrice: number | null
  evalAmount: number
}

interface Props {
  target: HoldingTarget | null
  accounts: AccountSummary[]
  manualAccounts: ManualAccount[]
  busy: boolean
  onClose: () => void
  onSubmitManual: (input: ManualInput) => void
  onSubmitFile: (holding: Holding, input: FileInput) => void
  onResetFile: (holding: Holding) => void
  /** 직접 추가한 종목을 편집할 때만 준다 */
  onRemoveManual?: () => void
}

export function HoldingForm({
  target,
  accounts,
  manualAccounts,
  busy,
  onClose,
  onSubmitManual,
  onSubmitFile,
  onResetFile,
  onRemoveManual,
}: Props) {
  const [name, setName] = useState('')
  const [category, setCategory] = useState<Category>(CATEGORIES[0])
  const [amount, setAmount] = useState('')
  const [accountId, setAccountId] = useState(NO_ACCOUNT)
  const [quantity, setQuantity] = useState('')
  const [avgBuyPrice, setAvgBuyPrice] = useState('')
  const [loaded, setLoaded] = useState<HoldingTarget | null>(null)

  // 열릴 때마다 대상에 맞춰 채운다.
  if (target !== loaded) {
    setLoaded(target)
    if (target) fill(target)
  }

  function fill(t: HoldingTarget) {
    if (t.kind === 'new') {
      setName('')
      setCategory(CATEGORIES[0])
      setAmount('')
      setAccountId(NO_ACCOUNT)
      return
    }
    if (t.kind === 'manual') {
      setName(t.item.name)
      setCategory(t.holding?.category ?? CATEGORIES[0])
      setAmount(commaFormat(String(t.item.evalAmount)))
      setAccountId(t.item.accountId ?? NO_ACCOUNT)
      return
    }
    setName(t.holding.name)
    setCategory(t.holding.category)
    setAmount(commaFormat(String(t.holding.evalAmount)))
    setQuantity(commaFormat(String(t.holding.quantity)))
    setAvgBuyPrice(t.holding.avgBuyPrice === null ? '' : commaFormat(String(t.holding.avgBuyPrice)))
  }

  function submit(event: React.FormEvent) {
    event.preventDefault()
    if (!target) return

    const evalAmount = commaParse(amount)
    if (!Number.isFinite(evalAmount) || evalAmount <= 0) return

    if (target.kind === 'file') {
      const qty = commaParse(quantity)
      if (!Number.isFinite(qty) || qty < 0) return
      onSubmitFile(target.holding, {
        category,
        quantity: qty,
        avgBuyPrice: avgBuyPrice.trim() === '' ? null : commaParse(avgBuyPrice),
        evalAmount,
      })
      return
    }

    if (!name.trim()) return
    onSubmitManual({
      name: name.trim(),
      category,
      evalAmount,
      accountId: accountId || undefined,
    })
  }

  const isFile = target?.kind === 'file'
  const edited = isFile && target.holding.original !== undefined
  const title = target?.kind === 'new' ? '종목 추가' : '종목 수정'

  return (
    <Modal open={target !== null} title={title} onClose={onClose}>
      <form className="modal-form" onSubmit={submit}>
        <label>
          <span>종목명</span>
          {isFile ? (
            <>
              <input value={name} disabled required />
              <small>잔고파일에서 온 종목이라 이름은 바꿀 수 없습니다.</small>
            </>
          ) : (
            <StockSearch
              value={name}
              onChange={(n) => setName(n)}
              disabled={busy}
            />
          )}
        </label>

        <label>
          <span>분류</span>
          <select
            value={category}
            onChange={(e) => setCategory(e.target.value as Category)}
            disabled={busy}
          >
            {CATEGORIES.map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </select>
        </label>

        {isFile && (
          <>
            <label>
              <span>보유수량</span>
              <input
                value={quantity}
                onChange={(e) => setQuantity(commaFormat(e.target.value))}
                inputMode="decimal"
                disabled={busy}
                required
              />
            </label>
            <label>
              <span>평균매입가</span>
              <input
                value={avgBuyPrice}
                onChange={(e) => setAvgBuyPrice(commaFormat(e.target.value))}
                inputMode="decimal"
                placeholder="비우면 손익을 계산하지 않습니다"
                disabled={busy}
              />
            </label>
          </>
        )}

        <label>
          <span>평가금액</span>
          <input
            value={amount}
            onChange={(e) => setAmount(commaFormat(e.target.value))}
            inputMode="numeric"
            disabled={busy}
            required
          />
          {isFile && <small>매입금액·손익·손익률·현재가는 여기서 다시 계산됩니다.</small>}
        </label>

        {!isFile && (
          <label>
            <span>소속 계좌</span>
            <select value={accountId} onChange={(e) => setAccountId(e.target.value)} disabled={busy}>
              <option value={NO_ACCOUNT}>계좌 없음</option>
              {accounts.filter((a) => !isManualAccountNumber(a.number)).map((a) => (
                <option key={a.number} value={a.number}>
                  {a.type} ({a.number})
                </option>
              ))}
              {manualAccounts.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.name}
                </option>
              ))}
            </select>
          </label>
        )}

        <footer className="modal-actions">
          {edited && (
            <button
              type="button"
              className="link danger"
              disabled={busy}
              onClick={() => onResetFile(target.holding)}
            >
              파일 값으로 되돌리기
            </button>
          )}
          {target?.kind === 'manual' && onRemoveManual && (
            <button type="button" className="link danger" disabled={busy} onClick={onRemoveManual}>
              삭제
            </button>
          )}
          <button type="button" className="modal-cancel" disabled={busy} onClick={onClose}>
            취소
          </button>
          <button type="submit" disabled={busy}>
            {target?.kind === 'new' ? '추가' : '수정 완료'}
          </button>
        </footer>
      </form>
    </Modal>
  )
}
