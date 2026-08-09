import { signedWon, won } from '../format'
import type { AccountSummary, Holding, ManualAccount } from '../types'
import { isManualAccountNumber, MANUAL_ACCOUNT_PREFIX } from '../types'

interface Props {
  accounts: AccountSummary[]
  holdings: Holding[]
  manualAccounts: ManualAccount[]
  superseded: ManualAccount[]
  coveredAsset: number
  busy: boolean
  onRemove: (accountNumber: string) => void
  onAddAccount: () => void
  onEditAccount: (id: string) => void
  onRemoveAccount: (id: string) => void
}

export function AccountsPanel({
  accounts,
  holdings,
  manualAccounts,
  superseded,
  coveredAsset,
  busy,
  onRemove,
  onAddAccount,
  onEditAccount,
  onRemoveAccount,
}: Props) {
  const profitByAccount = new Map<string, number>()
  for (const h of holdings) {
    if (h.profitLoss === null) continue
    profitByAccount.set(h.accountNumber, (profitByAccount.get(h.accountNumber) ?? 0) + h.profitLoss)
  }
  let totalProfit = 0
  for (const v of profitByAccount.values()) totalProfit += v
  return (
    <section className="accounts">
      <header className="section-head">
        <h2>계좌</h2>
        <button type="button" className="add-toggle" disabled={busy} onClick={onAddAccount}>
          계좌 추가
        </button>
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
            const id = account.number.slice(MANUAL_ACCOUNT_PREFIX.length)
            return (
              <li key={account.number} className={account.covered ? '' : 'excluded'}>
                <span className="account-type">
                  {account.type}
                  {accountLabel(account.number, manual, manualAccounts) && (
                    <span className="account-number">
                      {accountLabel(account.number, manual, manualAccounts)}
                    </span>
                  )}
                </span>
                <span className={`account-state ${account.covered ? 'on' : ''}`}>
                  {account.covered ? '집계됨' : '종목 없음'}
                </span>
                <span className="account-amount">
                  {won(account.totalAsset)}
                  {account.covered && profitByAccount.has(account.number) && (
                    <span className={profitByAccount.get(account.number)! >= 0 ? 'gain' : 'loss'}>
                      ({signedWon(profitByAccount.get(account.number)!)})
                    </span>
                  )}
                </span>
                {manual ? (
                  <span className="manual-actions">
                    <button
                      type="button"
                      className="link"
                      disabled={busy}
                      onClick={() => onEditAccount(id)}
                    >
                      수정
                    </button>
                    <button
                      type="button"
                      className="link"
                      disabled={busy}
                      onClick={() => onRemoveAccount(id)}
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
            <span className="account-amount">
              {won(coveredAsset)}
              {totalProfit !== 0 && (
                <span className={totalProfit >= 0 ? 'gain' : 'loss'}>
                  ({signedWon(totalProfit)})
                </span>
              )}
            </span>
            <span className="remove-placeholder" />
          </li>
        </ul>
      )}
    </section>
  )
}

function accountLabel(
  number: string,
  isManual: boolean,
  manualAccounts: ManualAccount[],
): string | null {
  if (!isManual) return number
  const id = number.slice(MANUAL_ACCOUNT_PREFIX.length)
  const acct = manualAccounts.find((a) => a.id === id)
  return acct?.accountNumber ?? null
}
