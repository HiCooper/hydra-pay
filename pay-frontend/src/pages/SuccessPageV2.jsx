import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { Spin } from 'antd'
import { CheckCircleFilled, LockOutlined } from '@ant-design/icons'
import { getCheckout } from '../api/checkout'
import { formatAmount } from '../utils'

export default function SuccessPageV2() {
  const { sessionId } = useParams()
  const [session, setSession] = useState(null)
  const [loading, setLoading] = useState(true)
  const [countdown, setCountdown] = useState(5)

  const params = new URLSearchParams(window.location.search)
  const embed = params.get('embed') === 'true'
  const parentOrigin = params.get('origin') || ''

  function postToParent(data) {
    if (embed && window.parent !== window.self) window.parent.postMessage(data, parentOrigin || '*')
  }

  useEffect(() => {
    getCheckout(sessionId)
      .then(data => { setSession(data); if (embed) postToParent({ type: 'hydra-pay:success', sessionId, amount: data.amount, status: 'paid' }) })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [sessionId])

  useEffect(() => {
    if (loading || embed) return
    const id = setInterval(() => {
      setCountdown(prev => {
        if (prev <= 1) { clearInterval(id); if (session?.success_url) window.location.href = session.success_url; return 0 }
        return prev - 1
      })
    }, 1000)
    return () => clearInterval(id)
  }, [loading, session, embed])

  if (loading) return <div style={S.page}><div style={S.card}><div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div></div></div>

  const amount = session ? formatAmount(session.amount) : '0.00'

  return (
    <div style={S.page}>
      {/* Left black panel */}
      <div style={{ flex: '0 0 50%', background: '#000' }} />

      {/* Right white panel */}
      <div style={{ flex: '0 0 50%', background: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <div style={{ maxWidth: 480, width: '100%', padding: '60px 48px', textAlign: 'center' }}>
          <div style={{
            width: 72, height: 72, background: '#04d66f', borderRadius: '50%',
            display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
            marginBottom: 24,
            animation: 'pop 0.4s cubic-bezier(0.34, 1.56, 0.64, 1)',
          }}>
            <CheckCircleFilled style={{ fontSize: 36, color: '#fff' }} />
          </div>

          <h2 style={{ fontSize: 22, fontWeight: 600, color: '#1a1a1a', margin: '0 0 10px', letterSpacing: '-0.3px' }}>支付成功</h2>
          <p style={{ fontSize: 28, fontWeight: 700, color: '#04d66f', margin: '0 0 28px', letterSpacing: '-0.5px' }}>¥ {amount}</p>

          <div style={{ height: 1, background: '#f0f0f0', margin: '0 0 24px' }} />

          {embed ? (
            <p style={{ fontSize: 14, color: '#6b6b6b', margin: 0 }}>请关闭此窗口</p>
          ) : (
            <>
              <p style={{ fontSize: 14, color: '#6b6b6b', margin: '0 0 4px' }}>
                {countdown > 0 ? `${countdown} 秒后自动跳转` : '正在跳转...'}
              </p>
              {session?.success_url && (
                <a href={session.success_url} style={{ color: '#f24b07', fontSize: 14, fontWeight: 500, textDecoration: 'none' }}>
                  立即跳转 →
                </a>
              )}
            </>
          )}
        </div>
      </div>

      <style>{`@keyframes pop{0%{transform:scale(0);opacity:0}60%{transform:scale(1.08)}100%{transform:scale(1);opacity:1}}`}</style>
    </div>
  )
}

const S = {
  page: {
    display: 'flex', minHeight: '100vh',
    fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
  },
  card: {
    background: '#fff', borderRadius: 12, border: '1px solid #e6e6e6',
    width: '100%', maxWidth: 420, margin: '80px auto',
  },
}
