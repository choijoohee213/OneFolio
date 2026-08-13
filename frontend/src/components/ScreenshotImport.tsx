import { useEffect, useRef, useState } from 'react'
import { extractFromScreenshot } from '../api'
import type { ExtractedHolding } from '../types'

interface Props {
  open: boolean
  busy: boolean
  onClose: () => void
  onConfirm: (holdings: ExtractedHolding[], accountNumber?: string) => void
}

type Step = 'upload' | 'preview' | 'extracting' | 'review'

function formatKRW(v: number | null): string {
  if (v == null) return ''
  return Math.round(v).toLocaleString('ko-KR')
}

function formatRate(v: number | null): string {
  if (v == null) return ''
  return v.toFixed(2)
}

function parseKRW(s: string): number | null {
  const raw = s.replace(/,/g, '')
  if (raw === '') return null
  const n = Number(raw)
  return isNaN(n) ? null : n
}

const MAX_DIMENSION = 1024

function resizeImage(file: File): Promise<File> {
  return new Promise((resolve) => {
    const img = new Image()
    img.onload = () => {
      const { width, height } = img
      if (width <= MAX_DIMENSION && height <= MAX_DIMENSION) {
        URL.revokeObjectURL(img.src)
        resolve(file)
        return
      }
      const scale = MAX_DIMENSION / Math.max(width, height)
      const canvas = document.createElement('canvas')
      canvas.width = Math.round(width * scale)
      canvas.height = Math.round(height * scale)
      canvas.getContext('2d')!.drawImage(img, 0, 0, canvas.width, canvas.height)
      URL.revokeObjectURL(img.src)
      canvas.toBlob(
        (blob) => resolve(new File([blob!], file.name, { type: 'image/jpeg' })),
        'image/jpeg',
        0.85,
      )
    }
    img.src = URL.createObjectURL(file)
  })
}

function useProgress(active: boolean) {
  const [pct, setPct] = useState(0)
  useEffect(() => {
    if (!active) { setPct(0); return }
    setPct(5)
    const t1 = setTimeout(() => setPct(20), 300)
    const t2 = setTimeout(() => setPct(40), 1200)
    const t3 = setTimeout(() => setPct(60), 3000)
    const t4 = setTimeout(() => setPct(75), 5000)
    const t5 = setTimeout(() => setPct(85), 8000)
    const t6 = setTimeout(() => setPct(92), 12000)
    return () => { clearTimeout(t1); clearTimeout(t2); clearTimeout(t3); clearTimeout(t4); clearTimeout(t5); clearTimeout(t6) }
  }, [active])
  return pct
}

export function ScreenshotImport({ open, busy, onClose, onConfirm }: Props) {
  const input = useRef<HTMLInputElement>(null)
  const [step, setStep] = useState<Step>('upload')
  const [holdings, setHoldings] = useState<ExtractedHolding[]>([])
  const [accountNumber, setAccountNumber] = useState<string>('')
  const [accountType, setAccountType] = useState<string>('')
  const [error, setError] = useState<string | null>(null)
  const [preview, setPreview] = useState<string | null>(null)
  const [imageFile, setImageFile] = useState<File | null>(null)
  const [showFullImage, setShowFullImage] = useState(false)
  const progress = useProgress(step === 'extracting')

  function reset() {
    setStep('upload')
    setHoldings([])
    setAccountNumber('')
    setAccountType('')
    setError(null)
    setShowFullImage(false)
    setImageFile(null)
    if (preview) {
      URL.revokeObjectURL(preview)
      setPreview(null)
    }
  }

  function handleClose() {
    reset()
    onClose()
  }

  function handleFile(file: File) {
    if (!file.type.startsWith('image/')) {
      setError('이미지 파일만 가능합니다')
      return
    }
    setError(null)
    setPreview(URL.createObjectURL(file))
    setImageFile(file)
    setStep('preview')
  }

  async function startExtract() {
    if (!imageFile) return
    setStep('extracting')
    try {
      const resized = await resizeImage(imageFile)
      const result = await extractFromScreenshot(resized)
      if (!result.holdings || result.holdings.length === 0) {
        setError('종목을 찾지 못했습니다. 다른 캡처를 시도해보세요.')
        setStep('preview')
        return
      }
      setHoldings(result.holdings)
      setAccountNumber(result.accountNumber ?? '')
      setAccountType(result.accountType ?? '')
      setStep('review')
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause))
      setStep('preview')
    }
  }

  function updateHolding(index: number, field: keyof ExtractedHolding, value: string) {
    setHoldings((prev) =>
      prev.map((h, i) => {
        if (i !== index) return h
        if (field === 'name') return { ...h, name: value }
        const num = parseKRW(value)
        if (value !== '' && num === null) return h
        return { ...h, [field]: num }
      }),
    )
  }

  function removeHolding(index: number) {
    setHoldings((prev) => prev.filter((_, i) => i !== index))
  }

  if (!open) return null

  const accountLabel = [accountNumber, accountType].filter(Boolean).join(' ')

  return (
    <dialog className="modal" open>
      <div className="modal-body">
        <header className="modal-head">
          <h3>
            {step === 'review'
              ? '추출 결과 확인'
              : step === 'preview'
                ? '이미지 확인'
                : '스크린샷으로 종목 추가'}
          </h3>
          <button type="button" className="modal-close" onClick={handleClose}>
            닫기
          </button>
        </header>

          <input
          ref={input}
          type="file"
          accept="image/*"
          hidden
          onChange={(e) => {
            const file = e.target.files?.[0]
            if (file) handleFile(file)
            e.target.value = ''
          }}
        />

        {step === 'upload' && (
          <div className="screenshot-upload">
            <div
              className="screenshot-drop"
              onDragOver={(e) => e.preventDefault()}
              onDrop={(e) => {
                e.preventDefault()
                const file = e.dataTransfer.files[0]
                if (file) handleFile(file)
              }}
            >
              <p>증권 앱 캡처를 여기에 끌어다 놓거나</p>
              <button type="button" onClick={() => input.current?.click()}>
                이미지 선택
              </button>
            </div>
            {error && <p className="screenshot-error">{error}</p>}
          </div>
        )}

        {step === 'preview' && (
          <div className="screenshot-preview-step">
            {preview && (
              <img
                src={preview}
                alt="캡처"
                className="screenshot-preview clickable"
                onClick={() => setShowFullImage(true)}
              />
            )}
            {error && <p className="screenshot-error">{error}</p>}
            <footer className="modal-actions">
              <button
                type="button"
                className="modal-cancel"
                onClick={() => {
                  setImageFile(null)
                  if (preview) URL.revokeObjectURL(preview)
                  setPreview(null)
                  setError(null)
                  input.current?.click()
                }}
              >
                재선택
              </button>
              <button type="button" className="modal-confirm" onClick={startExtract}>
                종목 추출
              </button>
            </footer>
          </div>
        )}

        {step === 'extracting' && (
          <div className="screenshot-extracting">
            {preview && <img src={preview} alt="캡처" className="screenshot-preview" />}
            <div className="extracting-loader">
              <div className="extracting-progress-wrap">
                <div className="extracting-progress-bar" style={{ width: `${progress}%` }} />
              </div>
              <p className="extracting-text">{progress}% · 종목을 읽고 있습니다...</p>
            </div>
          </div>
        )}

        {step === 'review' && (
          <div className="screenshot-review">
            {preview && (
              <img
                src={preview}
                alt="캡처"
                className="screenshot-preview clickable"
                onClick={() => setShowFullImage(true)}
              />
            )}
            {accountLabel && <p className="review-account">{accountLabel}</p>}
            <p className="review-hint">
              추출된 {holdings.length}개 종목을 확인하세요. 틀린 값은 직접 수정할 수 있습니다.
            </p>
            <div className="review-table-wrap">
              <table className="review-table">
                <thead>
                  <tr>
                    <th>종목명</th>
                    <th>수량</th>
                    <th>평가금액(원)</th>
                    <th>평균매입가(원)</th>
                    <th>평가손익(원)</th>
                    <th>손익률(%)</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {holdings.map((h, i) => (
                    <tr key={i}>
                      <td>
                        <input
                          value={h.name}
                          onChange={(e) => updateHolding(i, 'name', e.target.value)}
                        />
                        {h.ticker && <span className="review-ticker">{h.ticker}</span>}
                      </td>
                      <td>
                        <input
                          type="number"
                          value={h.quantity ?? ''}
                          onChange={(e) => updateHolding(i, 'quantity', e.target.value)}
                        />
                      </td>
                      <td>
                        <input
                          value={formatKRW(h.evalAmount)}
                          onChange={(e) => updateHolding(i, 'evalAmount', e.target.value)}
                        />
                      </td>
                      <td>
                        <input
                          value={formatKRW(h.avgBuyPrice)}
                          onChange={(e) => updateHolding(i, 'avgBuyPrice', e.target.value)}
                        />
                      </td>
                      <td>
                        <input
                          className={
                            h.profitLoss != null
                              ? h.profitLoss >= 0
                                ? 'profit'
                                : 'loss'
                              : undefined
                          }
                          value={formatKRW(h.profitLoss)}
                          onChange={(e) => updateHolding(i, 'profitLoss', e.target.value)}
                        />
                      </td>
                      <td>
                        <input
                          className={
                            h.profitRate != null
                              ? h.profitRate >= 0
                                ? 'profit'
                                : 'loss'
                              : undefined
                          }
                          value={formatRate(h.profitRate)}
                          onChange={(e) => updateHolding(i, 'profitRate', e.target.value)}
                        />
                      </td>
                      <td>
                        <button
                          type="button"
                          className="link danger"
                          onClick={() => removeHolding(i)}
                        >
                          삭제
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {error && <p className="screenshot-error">{error}</p>}
            <footer className="modal-actions">
              <button type="button" className="modal-cancel" onClick={handleClose}>
                취소
              </button>
              <button
                type="button"
                className="modal-confirm"
                disabled={busy || holdings.length === 0}
                onClick={() => {
                  onConfirm(holdings, accountNumber || undefined)
                  handleClose()
                }}
              >
                {holdings.length}개 종목 추가
              </button>
            </footer>
          </div>
        )}
      </div>

      {showFullImage && preview && (
        <div className="lightbox" onClick={() => setShowFullImage(false)}>
          <img src={preview} alt="캡처 원본" />
        </div>
      )}
    </dialog>
  )
}
