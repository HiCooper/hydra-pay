import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { Spin } from 'antd'
import { CheckCircleFilled, SafetyOutlined } from '@ant-design/icons'
import { getCheckout } from '../api/checkout'
import { formatAmount } from '../utils'

export default function SuccessPage() {
  const { sessionId } = useParams()
  const [session, setSession] = useState(null)
  const [loading, setLoading] = useState(true)
  const [countdown, setCountdown] = useState(5)

  const params = new URLSearchParams(window.location.search)
  const embed = params.get('embed') === 'true'
  const parentOrigin = params.get('origin') || ''

  function postToParent(data) {
    if (embed && window.parent !== window.self) {
      window.parent.postMessage(data, parentOrigin || '*')
    }
  }

  useEffect(() => {
    getCheckout(sessionId)
      .then(data => {
        setSession(data)
        if (embed) {
          postToParent({ type: 'hydra-pay:success', sessionId, amount: data.amount, currency: data.currency, status: 'paid' })
        }
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [sessionId])

  useEffect(() => {
    if (loading || embed) return
    const id = setInterval(() => {
      setCountdown(prev => {
        if (prev <= 1) {
          clearInterval(id)
          if (session?.success_url) {
            window.location.href = session.success_url
          }
          return 0
        }
        return prev - 1
      })
    }, 1000)
    return () => clearInterval(id)
  }, [loading, session, embed])

  if (loading) {
    return (
      <div style={embed ? styles.pageEmbed : styles.page}>
        <div style={styles.card}>
          <div style={{ textAlign: 'center', padding: 80 }}>
            <Spin size="large" />
          </div>
        </div>
      </div>
    )
  }

  const amount = session ? formatAmount(session.amount) : '0.00'

  return (
    <div style={embed ? styles.pageEmbed : styles.page}>
      <div style={{ ...styles.card, ...(embed ? styles.cardEmbed : {}) }}>
        <div style={{ padding: '60px 36px', textAlign: 'center' }}>
          {/* Success icon — Stripe green */}
          <div style={{
            width: 72, height: 72, background: '#04d66f', borderRadius: '50%',
            display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
            marginBottom: 24, animation: 'pop 0.4s ease',
          }}>
            <CheckCircleFilled style={{ fontSize: 36, color: '#fff' }} />
          </div>

          <h2 style={{
            fontSize: 22, fontWeight: 600, color: '#1a1a1a',
            margin: '0 0 8px', letterSpacing: '-0.2px',
          }}>
            支付成功
          </h2>

          <p style={{
            fontSize: 28, fontWeight: 700, color: '#04d66f',
            margin: '0 0 24px', letterSpacing: '-0.3px',
          }}>
            ¥ {amount}
          </p>

          {embed ? (
            <p style={{ fontSize: 14, color: '#6b6b6b', margin: 0 }}>请关闭此窗口</p>
          ) : (
            <>
              <p style={{ fontSize: 14, color: '#6b6b6b', margin: '0 0 16px' }}>
                {countdown > 0
                  ? `即将跳转回商户页面... ${countdown} 秒`
                  : '正在跳转...'}
              </p>
              {session?.success_url && (
                <a
                  href={session.success_url}
                  style={{
                    color: '#de481b', fontSize: 14, fontWeight: 500,
                    textDecoration: 'none', display: 'inline-block',
                  }}
                >
                  立即跳转 →
                </a>
              )}
            </>
          )}
        </div>
      </div>

      {!embed && (
        <p style={{
          textAlign: 'center', fontSize: 12, color: '#bbb', marginTop: 24,
          display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 4,
        }}>
          <SafetyOutlined style={{ fontSize: 11, color: '#04d66f' }} />
          Powered by HydraPay
        </p>
      )}

      <style>{`
        @keyframes pop {
          0% { transform: scale(0); opacity: 0; }
          60% { transform: scale(1.08); }
          100% { transform: scale(1); opacity: 1; }
        }
      `}</style>
    </div>
  )
}

const styles = {
  page: {
    minHeight: '100vh',
    background: '#f7f7f7',
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    padding: '40px 20px',
  },
  pageEmbed: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    padding: 0,
  },
  card: {
    background: '#fff',
    borderRadius: 12,
    boxShadow: '0 2px 24px rgba(0,0,0,0.08)',
    width: '100%',
    maxWidth: 420,
    border: '1px solid #e6e6e6',
  },
  cardEmbed: {
    borderRadius: 0,
    boxShadow: 'none',
    maxWidth: '100%',
    border: 'none',
  },
}
