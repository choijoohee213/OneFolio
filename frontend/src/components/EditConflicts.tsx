import { won } from '../format'
import type { FileValues, HoldingEdit, Summary } from '../types'
import { holdingKey } from '../types'

interface Props {
  conflicts: HoldingEdit[]
  summary: Summary
  busy: boolean
  onKeepMine: (edit: HoldingEdit, current: FileValues) => void
  onUseFile: (edit: HoldingEdit) => void
}

// 값을 고쳐 둔 종목인데 새로 올린 파일의 값이 달라졌을 때. 어느 쪽이 맞는지는
// 사용자만 아니까 미리 규칙을 정해 두지 않고 그때 물어본다.
export function EditConflicts({ conflicts, summary, busy, onKeepMine, onUseFile }: Props) {
  if (conflicts.length === 0) return null

  const current = new Map(
    summary.holdings
      .filter((h) => h.original)
      .map((h) => [holdingKey(h.accountNumber, h.name), h.original!]),
  )

  return (
    <section className="conflicts">
      <p className="notice warn">
        <strong>직접 고친 종목 {conflicts.length}개의 잔고파일 값이 달라졌습니다.</strong> 어느 쪽을
        쓸지 골라주세요.
      </p>
      <ul className="conflict-list">
        {conflicts.map((edit) => {
          const file = current.get(holdingKey(edit.accountNumber, edit.name))
          if (!file) return null
          return (
            <li key={holdingKey(edit.accountNumber, edit.name)}>
              <span className="conflict-name">{edit.name}</span>
              <span className="conflict-values">
                <span className="conflict-mine">내 값 {won(edit.evalAmount ?? file.evalAmount)}</span>
                <span className="conflict-file">새 파일 {won(file.evalAmount)}</span>
              </span>
              <span className="manual-actions">
                <button
                  type="button"
                  className="link"
                  disabled={busy}
                  onClick={() => onKeepMine(edit, file)}
                >
                  내 값 유지
                </button>
                <button type="button" className="link" disabled={busy} onClick={() => onUseFile(edit)}>
                  파일 값으로
                </button>
              </span>
            </li>
          )
        })}
      </ul>
    </section>
  )
}
