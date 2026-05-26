import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Spin, Modal, Button, message } from 'antd'
import { CheckCircleFilled, CloseCircleFilled, LeftOutlined, LockOutlined } from '@ant-design/icons'
import { getCheckout, activatePayment, getPaymentStatus } from '../api/checkout'
import { isMobile, formatAmount } from '../utils'
import ChannelCard from '../components/ChannelCard'
import QRCodeSection from '../components/QRCodeSection'
import CountdownTimer from '../components/CountdownTimer'
import { AlipayLogo, WechatPayLogo } from '../components/ChannelLogos'

export default function CheckoutPage() {
  const { sessionId } = useParams()
  const navigate = useNavigate()
  const [session, setSession] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [activating, setActivating] = useState(false)
  const [selectedChannel, setSelectedChannel] = useState(null)
  const [qrResult, setQrResult] = useState(null)
  const [paid, setPaid] = useState(false)

  // Embed mode detection
  const params = new URLSearchParams(window.location.search)
  const embed = params.get('embed') === 'true'
  const parentOrigin = params.get('origin') || ''

  function postToParent(data) {
    if (embed && window.parent !== window.self) {
      window.parent.postMessage(data, parentOrigin || '*')
    }
  }

  // Load session info
  useEffect(() => {
    getCheckout(sessionId)
      .then(data => { setSession(data); postToParent({ type: 'hydra-pay:ready', sessionId }) })
      .catch(err => { setError(err.message); postToParent({ type: 'hydra-pay:error', sessionId, message: err.message }) })
      .finally(() => setLoading(false))
  }, [sessionId])

  // Poll payment status once activated
  useEffect(() => {
    if (!qrResult) return
    const interval = setInterval(() => {
      getPaymentStatus(sessionId)
        .then(data => {
          if (data.status === 'paid') {
            clearInterval(interval)
            if (embed) {
              postToParent({ type: 'hydra-pay:success', sessionId, amount: session?.amount, currency: session?.currency, status: 'paid' })
            } else {
              setPaid(true)
              setTimeout(() => navigate(`/checkout/${sessionId}/success`), 1500)
            }
          }
        })
        .catch(() => {})
    }, 3000)
    return () => clearInterval(interval)
  }, [qrResult, sessionId, navigate])

  const mobile = isMobile()

  const handleChannelSelect = (channel) => {
    setSelectedChannel(channel)
  }

  const handlePay = async () => {
    if (!selectedChannel) return
    setActivating(true)
    try {
      const result = await activatePayment(sessionId, selectedChannel)
      setQrResult(result)
      if (mobile && result.payment_url) {
        if (embed) {
          postToParent({ type: 'hydra-pay:redirect', sessionId, url: result.payment_url })
        } else {
          window.location.href = result.payment_url
        }
      }
    } catch (err) {
      message.error(err.message || '支付激活失败')
    } finally {
      setActivating(false)
    }
  }

  // --- Render states ---

  if (loading) {
    return (
      <div style={embed ? styles.pageEmbed : styles.page}>
        <div style={{ ...styles.container, ...(embed ? styles.containerEmbed : {}) }}>
          <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div style={embed ? styles.pageEmbed : styles.page}>
        <div style={{ ...styles.container, ...(embed ? styles.containerEmbed : {}) }}>
          <div style={{ textAlign: 'center', padding: 60 }}>
            <CloseCircleFilled style={{ fontSize: 48, color: '#ff4d4f', marginBottom: 16 }} />
            <h3 style={{ marginBottom: 8, color: '#333' }}>页面加载失败</h3>
            <p style={{ color: '#999', fontSize: 14 }}>{error}</p>
          </div>
        </div>
      </div>
    )
  }

  if (paid) {
    return (
      <div style={embed ? styles.pageEmbed : styles.page}>
        <div style={{ ...styles.container, ...(embed ? styles.containerEmbed : {}) }}>
          <div style={{ textAlign: 'center', padding: 60 }}>
            <CheckCircleFilled style={{ fontSize: 48, color: '#52c41a', marginBottom: 16 }} />
            <h3 style={{ marginBottom: 8 }}>支付成功</h3>
            <p style={{ color: '#999', fontSize: 14 }}>即将跳转...</p>
          </div>
        </div>
      </div>
    )
  }

  // Server-reported terminal status — friendly pages matching Stripe UX
  if (session.status === 'expired') {
    postToParent({ type: 'hydra-pay:expired', sessionId })
    return <StatusPage icon={<CloseCircleFilled style={{ fontSize: 48, color: '#bfbfbf' }} />}
      title="该支付链接已过期"
      subtitle="请联系商户重新发起支付"
      merchantName={session.merchant_name}
      cancelUrl={session.cancel_url}
      embed={embed}
      onBack={embed ? () => postToParent({ type: 'hydra-pay:cancel', sessionId }) : null} />
  }

  if (session.status === 'completed') {
    postToParent({ type: 'hydra-pay:completed', sessionId })
    return <StatusPage icon={<CheckCircleFilled style={{ fontSize: 48, color: '#52c41a' }} />}
      title="该订单已支付完成"
      subtitle="如需帮助，请联系商户"
      merchantName={session.merchant_name}
      cancelUrl={session.cancel_url}
      embed={embed}
      onBack={embed ? () => postToParent({ type: 'hydra-pay:cancel', sessionId }) : null} />
  }

  const activeChannel = selectedChannel
  const merchantName = session.merchant_name || '商户'
  const backUrl = session.cancel_url || '#'

  return (
    <div style={embed ? styles.pageEmbed : styles.page}>
      <div style={{
        ...styles.container,
        ...(embed ? styles.containerEmbed : {}),
        ...(mobile ? styles.containerMobile : {}),
      }}>
        {/* ---- Left Column: Merchant + Payment ---- */}
        <div style={{
          ...styles.left,
          ...(mobile ? styles.leftMobile : {}),
        }}>
          {/* Back to merchant */}
          {embed ? (
            <button
              onClick={() => postToParent({ type: 'hydra-pay:cancel', sessionId })}
              style={{
                display: 'inline-flex', alignItems: 'center', gap: 4,
                fontSize: 13, color: '#666', textDecoration: 'none', marginBottom: 24,
                background: 'none', border: 'none', cursor: 'pointer', padding: 0,
              }}
            >
              <LeftOutlined style={{ fontSize: 11 }} />
              返回 {merchantName}
            </button>
          ) : (
            <a
              href={backUrl}
              style={{
                display: 'inline-flex', alignItems: 'center', gap: 4,
                fontSize: 13, color: '#666', textDecoration: 'none', marginBottom: 24,
              }}
            >
              <LeftOutlined style={{ fontSize: 11 }} />
              返回 {merchantName}
            </a>
          )}

          {/* Merchant name */}
          <h2 style={{ fontSize: 18, fontWeight: 600, color: '#333', margin: '0 0 6px' }}>
            {merchantName}
          </h2>

          {/* Description */}
          {session.description && (
            <p style={{ fontSize: 14, color: '#999', margin: '0 0 32px' }}>
              {session.description}
            </p>
          )}

          {/* Channel selection */}
          <div style={{ marginBottom: 20 }}>
            <div style={{ fontSize: 13, fontWeight: 500, color: '#333', marginBottom: 10 }}>
              选择支付方式
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
              <ChannelCard
                channel="alipay"
                active={selectedChannel === 'alipay'}
                onClick={() => handleChannelSelect('alipay')}
              />
              <ChannelCard
                channel="wechat"
                active={selectedChannel === 'wechat'}
                onClick={() => handleChannelSelect('wechat')}
              />
            </div>
          </div>

          {/* Pay Button */}
          <Button
            type="primary"
            block
            size="large"
            disabled={!selectedChannel}
            loading={activating}
            onClick={handlePay}
            style={{ height: 48, fontSize: 16, fontWeight: 600, borderRadius: 8 }}
          >
            去支付
          </Button>

        </div>

        {/* ---- Right Column: Order Summary / QR Code ---- */}
        {!mobile && (
          <div style={styles.right}>
            <div style={styles.summaryCard}>
              {/* Summary header */}
              <div style={{
                display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                marginBottom: 20, paddingBottom: 16,
                borderBottom: '1px solid #f0f0f0',
              }}>
                <span style={{ fontSize: 15, fontWeight: 600, color: '#333' }}>订单摘要</span>
                <LockOutlined style={{ fontSize: 13, color: '#52c41a' }} />
              </div>

              {/* Description */}
              {session.description && (
                <div style={{ fontSize: 13, color: '#666', marginBottom: 16 }}>
                  {session.description}
                </div>
              )}

              {/* Total */}
              <div style={{
                display: 'flex', alignItems: 'baseline', justifyContent: 'space-between',
                borderTop: '1px solid #f0f0f0', paddingTop: 16,
              }}>
                <span style={{ fontSize: 14, color: '#333', fontWeight: 500 }}>合计</span>
                <span style={{ fontSize: 22, fontWeight: 700, color: '#333' }}>
                  {session.currency === 'CNY' ? '¥' : session.currency} {formatAmount(session.amount)}
                </span>
              </div>

              {/* QR Code area — shown after activation */}
              {qrResult && (
                <div style={{ marginTop: 24, textAlign: 'center' }}>
                  <div style={{
                    borderTop: '1px solid #f0f0f0', paddingTop: 20, marginBottom: 12,
                  }}>
                    {/* Channel badge */}
                    <div style={{ marginBottom: 10 }}>
                      {activeChannel === 'wechat'
                        ? <WechatPayLogo height={24} />
                        : <AlipayLogo height={32} />
                      }
                    </div>
                    <p style={{ margin: '0 0 4px', fontSize: 13, color: '#666' }}>扫一扫付款</p>
                    <p style={{ margin: '0 0 16px', fontSize: 18, fontWeight: 700, color: '#333' }}>
                      ¥ {formatAmount(session.amount)} 元
                    </p>
                  </div>

                  <QRCodeSection
                    qrCodeURL={qrResult.qr_code_url}
                    paymentURL={qrResult.payment_url}
                  />

                  <div style={{ marginTop: 12 }}>
                    {session.expires_at && (
                      <p style={{ margin: '0 0 4px', fontSize: 12, color: '#999' }}>
                        二维码有效期 <CountdownTimer expiresAt={session.expires_at} inline />
                      </p>
                    )}
                    <p style={{ margin: 0, fontSize: 12, color: '#999' }}>请尽快完成付款</p>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}
      </div>

      {/* Footer branding */}
      {!embed && (
        <p style={{ textAlign: 'center', fontSize: 12, color: '#bbb', marginTop: 24 }}>
          Powered by HydraPay
        </p>
      )}

      {/* QR Code Modal — mobile only (desktop shows inline in right column) */}
      {mobile && (
        <Modal
          open={!!qrResult && !paid}
          onCancel={() => setQrResult(null)}
          footer={null}
          width={400}
          centered
          closable
        >
          <div style={{ padding: '32px 24px', textAlign: 'center' }}>
            <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'center' }}>
              {activeChannel === 'wechat'
                ? <WechatPayLogo height={28} />
                : <AlipayLogo height={40} />
              }
            </div>
            <p style={{ margin: '0 0 6px', fontSize: 13, color: '#666' }}>扫一扫付款</p>
            <p style={{ margin: '0 0 18px', fontSize: 22, fontWeight: 700, color: '#333' }}>
              ¥ {formatAmount(session.amount)} 元
            </p>
            <div style={{ marginBottom: 14 }}>
              <QRCodeSection
                qrCodeURL={qrResult?.qr_code_url}
                paymentURL={qrResult?.payment_url}
              />
            </div>
            {session.expires_at && (
              <p style={{ margin: '0 0 4px', fontSize: 12, color: '#999' }}>
                二维码有效期 <CountdownTimer expiresAt={session.expires_at} inline />
              </p>
            )}
            <p style={{ margin: 0, fontSize: 12, color: '#999' }}>请尽快完成付款</p>
          </div>
        </Modal>
      )}
    </div>
  )
}

// StatusPage — friendly terminal-state page for expired/completed sessions.
function StatusPage({ icon, title, subtitle, merchantName, cancelUrl, embed, onBack }) {
  return (
    <div style={embed ? styles.pageEmbed : styles.page}>
      <div style={{
        background: '#fff',
        borderRadius: 12,
        boxShadow: embed ? 'none' : '0 2px 24px rgba(0,0,0,0.08)',
        width: '100%',
        maxWidth: 420,
        padding: '60px 40px',
        textAlign: 'center',
      }}>
        <div style={{ marginBottom: 16 }}>{icon}</div>
        <h3 style={{ margin: '0 0 8px', fontSize: 18, color: '#333' }}>{title}</h3>
        <p style={{ margin: '0 0 24px', fontSize: 14, color: '#999' }}>{subtitle}</p>
        {embed && onBack ? (
          <button
            onClick={onBack}
            style={{
              display: 'inline-block', padding: '10px 32px',
              background: '#1677ff', color: '#fff', borderRadius: 8,
              fontSize: 14, fontWeight: 500, border: 'none', cursor: 'pointer',
            }}
          >
            返回{merchantName || '商户'}
          </button>
        ) : cancelUrl ? (
          <a
            href={cancelUrl}
            style={{
              display: 'inline-block', padding: '10px 32px',
              background: '#1677ff', color: '#fff', borderRadius: 8,
              fontSize: 14, fontWeight: 500, textDecoration: 'none',
            }}
          >
            返回{merchantName || '商户'}
          </a>
        ) : null}
      </div>
      {!embed && (
        <p style={{ textAlign: 'center', fontSize: 12, color: '#bbb', marginTop: 24 }}>
          Powered by HydraPay
        </p>
      )}
    </div>
  )
}

const styles = {
  page: {
    minHeight: '100vh',
    background: '#f0f2f5',
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
  container: {
    display: 'flex',
    background: '#fff',
    borderRadius: 12,
    boxShadow: '0 2px 24px rgba(0,0,0,0.08)',
    width: '100%',
    maxWidth: 780,
    overflow: 'hidden',
  },
  containerEmbed: {
    borderRadius: 0,
    boxShadow: 'none',
    maxWidth: '100%',
  },
  containerMobile: {
    flexDirection: 'column',
    maxWidth: 420,
  },
  left: {
    flex: '1 1 55%',
    padding: '40px 40px 40px 44px',
  },
  leftMobile: {
    padding: '32px 24px',
  },
  right: {
    flex: '0 0 45%',
    background: '#fafafa',
    borderLeft: '1px solid #f0f0f0',
    display: 'flex',
    alignItems: 'center',
  },
  summaryCard: {
    padding: 36,
    width: '100%',
  },
}
