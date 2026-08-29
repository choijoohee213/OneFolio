import { useState } from 'react'
import { Modal } from './Modal'
import { ThemeToggle } from './ThemeToggle'
import type { Theme } from '../theme'

interface Props {
  theme: Theme
  canReset: boolean
  onThemeChange: (theme: Theme) => void
  onReset: () => void
}

// 자주 안 건드리는 것(테마)과 위험한 것(초기화)을 헤더 밖으로 빼서 한곳에 모은다.
// 초기화는 되돌릴 수 없어, 같은 창을 확인 화면으로 바꿔 한 번 더 묻는다.
export function SettingsMenu({ theme, canReset, onThemeChange, onReset }: Props) {
  const [open, setOpen] = useState(false)
  const [confirming, setConfirming] = useState(false)

  function close() {
    setOpen(false)
    setConfirming(false)
  }

  return (
    <>
      <button type="button" className="settings-trigger" aria-label="설정" onClick={() => setOpen(true)}>
        <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
          <path
            fill="none"
            stroke="currentColor"
            strokeWidth="1.7"
            strokeLinecap="round"
            strokeLinejoin="round"
            d="M12 15.2a3.2 3.2 0 1 0 0-6.4 3.2 3.2 0 0 0 0 6.4Zm7.6-3.2c0 .5-.05 1-.13 1.47l2.03 1.55-1.9 3.3-2.4-.95c-.74.63-1.6 1.12-2.53 1.44L14.3 22h-3.8l-.37-2.19a7.7 7.7 0 0 1-2.53-1.44l-2.4.95-1.9-3.3 2.03-1.55a7.9 7.9 0 0 1 0-2.94L3.3 9.98l1.9-3.3 2.4.95A7.7 7.7 0 0 1 10.13 6.2L10.5 4h3.8l.37 2.19c.93.32 1.79.81 2.53 1.44l2.4-.95 1.9 3.3-2.03 1.55c.08.48.13.97.13 1.47Z"
          />
        </svg>
      </button>

      <Modal
        open={open}
        title={confirming ? '전부 초기화할까요?' : '설정'}
        className="settings-modal"
        onClose={close}
      >
        {confirming ? (
          <div className="settings-confirm">
            <p>
              올린 잔고파일, 직접 추가한 계좌·종목, 분류와 수정 내역이 모두 지워지고 첫 화면으로
              돌아갑니다. <strong>되돌릴 수 없습니다.</strong>
            </p>
            <div className="modal-actions">
              <button type="button" className="modal-cancel" onClick={() => setConfirming(false)}>
                취소
              </button>
              <button
                type="button"
                className="settings-danger"
                onClick={() => {
                  onReset()
                  close()
                }}
              >
                지우고 첫 화면으로
              </button>
            </div>
          </div>
        ) : (
          <div className="settings-sections">
            <section>
              <h4>화면 테마</h4>
              <ThemeToggle theme={theme} onChange={onThemeChange} />
            </section>
            {canReset && (
              <section>
                <h4>데이터</h4>
                <p className="settings-note">
                  지우면 이 브라우저에 저장된 내 자산 정보가 모두 사라집니다.
                </p>
                <button type="button" className="settings-danger" onClick={() => setConfirming(true)}>
                  전부 초기화
                </button>
              </section>
            )}
          </div>
        )}
      </Modal>
    </>
  )
}
