import { won } from '../format'
import type { AccountSummary } from '../types'

interface Props {
  accounts: AccountSummary[]
  coveredAsset: number
  busy: boolean
  onRemove: (accountNumber: string) => void
}

export function AccountsPanel({ accounts, coveredAsset, busy, onRemove }: Props) {
  return (
    <section className="accounts">
      <ul className="account-list">
        {accounts.map((account) => (
          <li key={account.number} className={account.covered ? '' : 'excluded'}>
            <span className="account-type">{account.type}</span>
            <span className={`account-state ${account.covered ? 'on' : ''}`}>
              {account.covered ? '집계됨' : '종목 없음'}
            </span>
            <span className="account-amount">{won(account.totalAsset)}</span>
            {account.covered ? (
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
        ))}
        <li className="account-sum">
          <span className="account-type">집계 기준</span>
          <span className="account-state" />
          <span className="account-amount">{won(coveredAsset)}</span>
          <span className="remove-placeholder" />
        </li>
      </ul>
    </section>
  )
}
