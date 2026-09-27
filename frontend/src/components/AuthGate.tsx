import { useEffect, useState } from 'react'
import { onUnauthorized, verifyToken } from '../api'
import { clearToken, loadToken, saveToken } from '../auth'

/** 암호가 없으면 앱을 아예 띄우지 않는다. 앱이 먼저 뜨면 저장된 잔고파일로
 *  집계 요청이 나가 401 로 깨지고, 암호를 넣은 뒤에도 그 화면이 그대로 남는다. */
export function AuthGate({ children }: { children: React.ReactNode }) {
  const [token, setToken] = useState<string | null>(loadToken)

  useEffect(() => {
    onUnauthorized(() => {
      clearToken()
      setToken(null)
    })
  }, [])

  if (token === null) {
    return (
      <PasswordPrompt
        onPass={(value) => {
          saveToken(value)
          setToken(value)
        }}
      />
    )
  }
  return <>{children}</>
}

function PasswordPrompt({ onPass }: { onPass: (token: string) => void }) {
  const [value, setValue] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    const token = value.trim()
    if (!token || busy) return

    setBusy(true)
    setError(null)
    try {
      if (await verifyToken(token)) onPass(token)
      else setError('암호가 맞지 않습니다.')
    } catch {
      setError('서버에 연결하지 못했습니다. 잠시 후 다시 시도해 주세요.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <main className="gate">
      <form className="gate-card" onSubmit={submit}>
        <h1>
          <img className="brand-mark" src="/favicon.svg" alt="" />
          OneFolio
        </h1>
        <label>
          암호
          <input
            type="password"
            value={value}
            autoFocus
            autoComplete="current-password"
            disabled={busy}
            onChange={(event) => setValue(event.target.value)}
          />
        </label>
        <button type="submit" disabled={busy || value.trim() === ''}>
          {busy ? '확인 중…' : '들어가기'}
        </button>
        {/* 무료 티어 서버는 한동안 아무도 안 부르면 잠든다. 처음 한 번은 깨어나기를
            기다려야 해서, 안 눌린 것처럼 보이는 시간이 길다. */}
        {busy && <small>서버가 잠들어 있으면 처음 한 번은 시간이 걸립니다.</small>}
        {error && <p className="gate-error">{error}</p>}
      </form>
    </main>
  )
}
