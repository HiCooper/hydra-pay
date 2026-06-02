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
      color: { dark: '#1a1a1a', light: '#ffffff' },
    })
  }, [code])

  if (!code) {
    return (
      <div style={{
        width: 220, height: 220, background: '#f7f7f7', borderRadius: 8,
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
        padding: 14,
        border: '1px solid #e6e6e6',
        borderRadius: 10,
        background: '#fff',
      }}
    />
  )
}
