import { won } from '../format'
import type { AccountSummary } from '../types'

interface Props {
  accounts: AccountSummary[]
  coveredAsset: number
  totalAsset: number
}

export function AccountsPanel({ accounts, coveredAsset, totalAsset }: Props) {
  const missing = accounts.filter((account) => !account.covered)

  return (
    <section className="accounts">
      {missing.length > 0 && (
        <p className="notice">
          <strong>계좌 {missing.length}개의 종목 정보가 없습니다.</strong> 아래 배분과 비중은 종목을
          올린 계좌({won(coveredAsset)})만 기준입니다. 전체 자산 배분을 보려면{' '}
          {missing.map((account) => account.type).join(', ')} 파일도 같이 올리세요.
        </p>
      )}

      <ul className="account-list">
        {accounts.map((account) => (
          <li key={account.number} className={account.covered ? '' : 'excluded'}>
            <span className="account-type">{account.type}</span>
            <span className="account-state">{account.covered ? '집계됨' : '종목 없음'}</span>
            <span className="account-amount">{won(account.totalAsset)}</span>
          </li>
        ))}
        <li className="account-sum">
          <span className="account-type">전체 계좌 합</span>
          <span className="account-state" />
          <span className="account-amount">{won(totalAsset)}</span>
        </li>
      </ul>
    </section>
  )
}
