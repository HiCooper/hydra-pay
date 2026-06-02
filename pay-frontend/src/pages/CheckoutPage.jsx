import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Spin, Modal, message } from 'antd'
import { CheckCircleFilled, CloseCircleFilled, LeftOutlined, LockOutlined, SafetyOutlined } from '@ant-design/icons'
import { getCheckout, activatePayment, getPaymentStatus } from '../api/checkout'
import { isMobile, formatAmount } from '../utils'
import ChannelCard from '../components/ChannelCard'
import QRCodeSection from '../components/QRCodeSection'
import CountdownTimer from '../components/CountdownTimer'
import { AlipayLogo, WechatPayLogo, UnionpayLogo } from '../components/ChannelLogos'

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
      // 云闪付：桌面走原生扫码(native)，移动走H5跳转
      const tradeType = selectedChannel === 'unionpay' ? (mobile ? 'h5' : 'native') : undefined
      const result = await activatePayment(sessionId, selectedChannel, tradeType)
      setQrResult(result)
      if (mobile && result.payment_url) {
        if (selectedChannel === 'unionpay') {
          // H5 返回的是完整 HTML，用 Blob URL 打开
          const blob = new Blob([result.payment_url], { type: 'text/html' })
          const url = URL.createObjectURL(blob)
          if (embed) {
            postToParent({ type: 'hydra-pay:redirect', sessionId, url })
          } else {
            window.location.href = url
          }
        } else if (embed) {
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
      <div style={embed ? s.pageEmbed : s.page}>
        <div style={{ ...s.card, ...(embed ? s.cardEmbed : {}) }}>
          <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div style={embed ? s.pageEmbed : s.page}>
        <div style={{ ...s.card, ...(embed ? s.cardEmbed : {}) }}>
          <div style={{ textAlign: 'center', padding: 60 }}>
            <CloseCircleFilled style={{ fontSize: 48, color: '#df1b41', marginBottom: 16 }} />
            <h3 style={{ marginBottom: 8, color: '#1a1a1a', fontSize: 18, fontWeight: 600 }}>页面加载失败</h3>
            <p style={{ color: '#6b6b6b', fontSize: 14 }}>{error}</p>
          </div>
        </div>
      </div>
    )
  }

  if (paid) {
    return (
      <div style={embed ? s.pageEmbed : s.page}>
        <div style={{ ...s.card, ...(embed ? s.cardEmbed : {}) }}>
          <div style={{ textAlign: 'center', padding: 60 }}>
            <CheckCircleFilled style={{ fontSize: 48, color: '#04d66f', marginBottom: 16 }} />
            <h3 style={{ marginBottom: 8, color: '#1a1a1a', fontSize: 18, fontWeight: 600 }}>支付成功</h3>
            <p style={{ color: '#6b6b6b', fontSize: 14 }}>即将跳转...</p>
          </div>
        </div>
      </div>
    )
  }

  // Server-reported terminal status
  if (session.status === 'expired') {
    postToParent({ type: 'hydra-pay:expired', sessionId })
    return <StatusPage icon={<CloseCircleFilled style={{ fontSize: 48, color: '#bbb' }} />}
      title="该支付链接已过期"
      subtitle="请联系商户重新发起支付"
      merchantName={session.merchant_name}
      cancelUrl={session.cancel_url}
      embed={embed}
      onBack={embed ? () => postToParent({ type: 'hydra-pay:cancel', sessionId }) : null} />
  }

  if (session.status === 'completed') {
    postToParent({ type: 'hydra-pay:completed', sessionId })
    return <StatusPage icon={<CheckCircleFilled style={{ fontSize: 48, color: '#04d66f' }} />}
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
    <div style={embed ? s.pageEmbed : s.page}>
      <div style={{
        ...s.card,
        ...(embed ? s.cardEmbed : {}),
        ...(mobile ? s.cardMobile : {}),
      }}>

        {/* ====== Left Column ====== */}
        <div style={{ ...s.left, ...(mobile ? s.leftMobile : {}) }}>

          {/* Back link */}
          {embed ? (
            <button onClick={() => postToParent({ type: 'hydra-pay:cancel', sessionId })} style={s.backLink}>
              <LeftOutlined style={{ fontSize: 11 }} />
              返回 {merchantName}
            </button>
          ) : (
            <a href={backUrl} style={s.backLink}>
              <LeftOutlined style={{ fontSize: 11 }} />
              返回 {merchantName}
            </a>
          )}

          {/* Trust badge — Stripe green */}
          <div style={s.trustBadge}>
            <SafetyOutlined style={{ fontSize: 13, color: '#04d66f', marginRight: 5 }} />
            <span>安全支付</span>
          </div>

          {/* Merchant name */}
          <h2 style={s.merchantName}>{merchantName}</h2>

          {/* Description */}
          {session.description && (
            <p style={s.description}>{session.description}</p>
          )}

          {/* Divider */}
          <div style={s.divider} />

          {/* Payment method selection */}
          <div style={{ marginBottom: 24 }}>
            <div style={s.sectionLabel}>选择支付方式</div>
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
              <ChannelCard
                channel="unionpay"
                active={selectedChannel === 'unionpay'}
                onClick={() => handleChannelSelect('unionpay')}
              />
            </div>
          </div>

          {/* Pay button — Stripe orange */}
          <button
            disabled={!selectedChannel || activating || !!qrResult}
            onClick={handlePay}
            style={{
              ...s.payButton,
              ...((!selectedChannel || !!qrResult) ? s.payButtonDisabled : {}),
              ...(activating ? s.payButtonLoading : {}),
            }}
          >
            {activating ? (
              <span style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8 }}>
                <Spin size="small" style={{ color: '#fff' }} />
                处理中...
              </span>
            ) : qrResult ? (
              `扫码支付中 · ¥${formatAmount(session.amount)}`
            ) : (
              `支付 ¥${formatAmount(session.amount)}`
            )}
          </button>

          {/* Footer text */}
          <p style={s.footerText}>
            支付即表示您同意服务条款
          </p>
        </div>

        {/* ====== Right Column: Order Summary ====== */}
        {!mobile && (
          <div style={s.right}>
            <div style={s.summaryCard}>

              {/* Summary header with lock */}
              <div style={s.summaryHeader}>
                <span style={s.summaryTitle}>订单摘要</span>
                <div style={s.lockBadge}>
                  <LockOutlined style={{ fontSize: 12, color: '#04d66f' }} />
                </div>
              </div>

              {/* Merchant */}
              <div style={s.summaryRow}>
                <span style={s.summaryLabel}>商户</span>
                <span style={s.summaryValue}>{merchantName}</span>
              </div>

              {/* Description */}
              {session.description && (
                <div style={s.summaryRow}>
                  <span style={s.summaryLabel}>商品</span>
                  <span style={s.summaryValue}>{session.description}</span>
                </div>
              )}

              {/* Divider */}
              <div style={s.summaryDivider} />

              {/* Total */}
              <div style={s.summaryTotal}>
                <span style={s.summaryLabel}>合计支付</span>
                <span style={s.summaryAmount}>
                  {session.currency === 'CNY' ? '¥' : session.currency} {formatAmount(session.amount)}
                </span>
              </div>

              {/* QR Code area — shown after activation */}
              {qrResult && (
                <div style={{ marginTop: 24 }}>
                  <div style={s.summaryDivider} />

                  <div style={{ textAlign: 'center', marginTop: 20 }}>
                    {/* Channel badge */}
                    <div style={{ marginBottom: 10 }}>
                      {activeChannel === 'wechat'
                        ? <WechatPayLogo height={24} />
                        : activeChannel === 'unionpay'
                        ? <UnionpayLogo height={28} />
                        : <AlipayLogo height={32} />
                      }
                    </div>
                    <p style={s.qrLabel}>扫一扫付款</p>
                    <p style={s.qrAmount}>
                      ¥ {formatAmount(session.amount)}
                    </p>

                    <QRCodeSection
                      qrCodeURL={qrResult.qr_code_url}
                      paymentURL={qrResult.payment_url}
                    />

                    <div style={{ marginTop: 14 }}>
                      {session.expires_at && (
                        <p style={{ margin: '0 0 4px', fontSize: 12, color: '#999' }}>
                          二维码有效期 <CountdownTimer expiresAt={session.expires_at} inline />
                        </p>
                      )}
                      <p style={{ margin: 0, fontSize: 12, color: '#999' }}>请尽快完成付款</p>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}
      </div>

      {/* Footer branding */}
      {!embed && (
        <p style={s.brandFooter}>
          <SafetyOutlined style={{ fontSize: 11, marginRight: 4, color: '#04d66f' }} />
          Powered by HydraPay
        </p>
      )}

      {/* QR Code Modal — mobile only */}
      {mobile && (
        <Modal
          open={!!qrResult && !paid}
          onCancel={() => setQrResult(null)}
          footer={null}
          width={400}
          centered
          closable
        >
          <div style={{ padding: '28px 20px', textAlign: 'center' }}>
            <div style={{ marginBottom: 10, display: 'flex', justifyContent: 'center' }}>
              {activeChannel === 'wechat'
                ? <WechatPayLogo height={28} />
                : activeChannel === 'unionpay'
                ? <UnionpayLogo height={36} />
                : <AlipayLogo height={40} />
              }
            </div>
            <p style={s.qrLabel}>扫一扫付款</p>
            <p style={s.qrAmount}>
              ¥ {formatAmount(session.amount)}
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

// --- StatusPage ---
function StatusPage({ icon, title, subtitle, merchantName, cancelUrl, embed, onBack }) {
  return (
    <div style={embed ? s.pageEmbed : s.page}>
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
        <h3 style={{ margin: '0 0 8px', fontSize: 18, fontWeight: 600, color: '#1a1a1a' }}>{title}</h3>
        <p style={{ margin: '0 0 24px', fontSize: 14, color: '#6b6b6b' }}>{subtitle}</p>
        {embed && onBack ? (
          <button onClick={onBack} style={s.statusButton}>
            返回{merchantName || '商户'}
          </button>
        ) : cancelUrl ? (
          <a href={cancelUrl} style={{ ...s.statusButton, textDecoration: 'none', display: 'inline-block' }}>
            返回{merchantName || '商户'}
          </a>
        ) : null}
      </div>
      {!embed && (
        <p style={s.brandFooter}>
          <SafetyOutlined style={{ fontSize: 11, marginRight: 4, color: '#04d66f' }} />
          Powered by HydraPay
        </p>
      )}
    </div>
  )
}

// ====== Styles ======
const s = {
  // Page
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

  // Main card — two-column layout
  card: {
    display: 'flex',
    background: '#fff',
    borderRadius: 12,
    boxShadow: '0 2px 24px rgba(0,0,0,0.08)',
    width: '100%',
    maxWidth: 780,
    overflow: 'hidden',
    border: '1px solid #e6e6e6',
  },
  cardEmbed: {
    borderRadius: 0,
    boxShadow: 'none',
    maxWidth: '100%',
    border: 'none',
  },
  cardMobile: {
    flexDirection: 'column',
    maxWidth: 420,
  },

  // Left column
  left: {
    flex: '1 1 55%',
    padding: '44px 40px 40px 48px',
  },
  leftMobile: {
    padding: '32px 28px',
  },

  // Back link
  backLink: {
    display: 'inline-flex',
    alignItems: 'center',
    gap: 4,
    fontSize: 13,
    color: '#6b6b6b',
    textDecoration: 'none',
    marginBottom: 20,
    background: 'none',
    border: 'none',
    cursor: 'pointer',
    padding: 0,
  },

  // Trust badge — Stripe green style
  trustBadge: {
    display: 'inline-flex',
    alignItems: 'center',
    gap: 4,
    fontSize: 12,
    fontWeight: 500,
    color: '#04d66f',
    background: '#f0fdf6',
    padding: '4px 10px',
    borderRadius: 4,
    marginBottom: 20,
  },

  // Merchant name
  merchantName: {
    fontSize: 22,
    fontWeight: 600,
    color: '#1a1a1a',
    margin: '0 0 8px',
    letterSpacing: '-0.3px',
  },

  // Description
  description: {
    fontSize: 14,
    color: '#6b6b6b',
    margin: '0 0 0 0',
    lineHeight: 1.5,
  },

  // Section label
  sectionLabel: {
    fontSize: 13,
    fontWeight: 500,
    color: '#1a1a1a',
    marginBottom: 12,
  },

  // Divider
  divider: {
    height: 1,
    background: '#f0f0f0',
    margin: '24px 0',
  },

  // Pay button — Stripe orange
  payButton: {
    width: '100%',
    height: 48,
    background: '#de481b',
    color: '#fff',
    border: 'none',
    borderRadius: 6,
    fontSize: 16,
    fontWeight: 600,
    cursor: 'pointer',
    transition: 'all 0.15s ease',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  },
  payButtonDisabled: {
    background: '#e6e6e6',
    color: '#999',
    cursor: 'not-allowed',
  },
  payButtonLoading: {
    opacity: 0.85,
    cursor: 'wait',
  },

  // Footer text
  footerText: {
    textAlign: 'center',
    color: '#bbb',
    fontSize: 11,
    marginTop: 16,
    marginBottom: 0,
  },

  // Right column — order summary
  right: {
    flex: '0 0 45%',
    background: '#fafafa',
    borderLeft: '1px solid #f0f0f0',
    display: 'flex',
    alignItems: 'flex-start',
  },
  summaryCard: {
    padding: 40,
    width: '100%',
  },

  // Summary header
  summaryHeader: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 24,
    paddingBottom: 16,
    borderBottom: '1px solid #f0f0f0',
  },
  summaryTitle: {
    fontSize: 15,
    fontWeight: 600,
    color: '#1a1a1a',
  },
  lockBadge: {
    width: 28,
    height: 28,
    background: '#f0fdf6',
    borderRadius: '50%',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  },

  // Summary rows
  summaryRow: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 12,
  },
  summaryLabel: {
    fontSize: 13,
    color: '#6b6b6b',
  },
  summaryValue: {
    fontSize: 13,
    color: '#1a1a1a',
    fontWeight: 500,
    textAlign: 'right',
    maxWidth: '60%',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },

  // Summary divider
  summaryDivider: {
    height: 1,
    background: '#f0f0f0',
    margin: '4px 0 16px',
  },

  // Total
  summaryTotal: {
    display: 'flex',
    alignItems: 'baseline',
    justifyContent: 'space-between',
  },
  summaryAmount: {
    fontSize: 22,
    fontWeight: 700,
    color: '#1a1a1a',
    letterSpacing: '-0.3px',
  },

  // QR area
  qrLabel: {
    margin: '0 0 4px',
    fontSize: 13,
    color: '#6b6b6b',
  },
  qrAmount: {
    margin: '0 0 16px',
    fontSize: 20,
    fontWeight: 700,
    color: '#1a1a1a',
  },

  // Brand footer
  brandFooter: {
    textAlign: 'center',
    fontSize: 12,
    color: '#bbb',
    marginTop: 24,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  },

  // Status page button
  statusButton: {
    display: 'inline-block',
    padding: '10px 32px',
    background: '#de481b',
    color: '#fff',
    borderRadius: 6,
    fontSize: 14,
    fontWeight: 500,
    border: 'none',
    cursor: 'pointer',
  },
}
