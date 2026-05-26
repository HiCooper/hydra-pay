import { useEffect, useRef } from 'react'
import QRCode from 'qrcode'

export default function QRCodeSection({ qrCodeURL, paymentURL }) {
  const canvasRef = useRef(null)
  const code = qrCodeURL || paymentURL

  useEffect(() => {
    if (!code || !canvasRef.current) return
    QRCode.toCanvas(canvasRef.current, code, {
      width: 220,
      margin: 2,
      color: { dark: '#1a1a2e', light: '#ffffff' },
    })
  }, [code])

  if (!code) {
    return (
      <div style={{
        width: 220, height: 220, background: '#f5f5f5', borderRadius: 8,
        display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
        color: '#bbb', fontSize: 13,
      }}>
        二维码暂不可用
      </div>
    )
  }

  return (
    <canvas
      ref={canvasRef}
      style={{
        display: 'inline-block',
        padding: 16,
        border: '2px solid #f0f0f0',
        borderRadius: 12,
        background: '#fafafa',
      }}
    />
  )
}
