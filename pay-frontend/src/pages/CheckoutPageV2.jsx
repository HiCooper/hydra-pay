import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Spin, Modal, message } from 'antd'
import { CheckCircleFilled, CloseCircleFilled, LeftOutlined, LockOutlined } from '@ant-design/icons'
import { getCheckout, activatePayment, getPaymentStatus } from '../api/checkout'
import { isMobile, formatAmount } from '../utils'
import QRCodeSection from '../components/QRCodeSection'
import CountdownTimer from '../components/CountdownTimer'
import { AlipayLogo, WechatPayLogo, UnionpayLogo } from '../components/ChannelLogos'

/* =====================================================================
   CheckoutPageV2 — 1:1 Stripe Checkout clone

   Based on the subscription checkout screenshot (2698×1380):
   - True 50/50 split: LEFT = black panel, RIGHT = white checkout form
   - Green trust badge (#04d66f) at top of right panel
   - Orange CTA button (#f24b07)
   - Gray form sections (#f7f7f7), hairline dividers (#e6e6e6)
   ===================================================================== */

export default function CheckoutPageV2() {
  const { sessionId } = useParams()
  const navigate = useNavigate()
  const [session, setSession] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [activating, setActivating] = useState(false)
  const [selectedChannel, setSelectedChannel] = useState(null)
  const [qrResult, setQrResult] = useState(null)
  const [paid, setPaid] = useState(false)

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
      .then(data => { setSession(data); postToParent({ type: 'hydra-pay:ready', sessionId }) })
      .catch(err => { setError(err.message); postToParent({ type: 'hydra-pay:error', sessionId, message: err.message }) })
      .finally(() => setLoading(false))
  }, [sessionId])

  useEffect(() => {
    if (!qrResult) return
    const t = setInterval(() => {
      getPaymentStatus(sessionId).then(data => {
        if (data.status === 'paid') {
          clearInterval(t)
          if (embed) postToParent({ type: 'hydra-pay:success', sessionId, amount: session?.amount, status: 'paid' })
          else { setPaid(true); setTimeout(() => navigate('/v2/checkout/' + sessionId + '/success'), 1500) }
        }
      }).catch(() => {})
    }, 3000)
    return () => clearInterval(t)
  }, [qrResult, sessionId, navigate])

  const mobile = isMobile()

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
          if (embed) postToParent({ type: 'hydra-pay:redirect', sessionId, url })
          else window.location.href = url
        } else if (embed) postToParent({ type: 'hydra-pay:redirect', sessionId, url: result.payment_url })
        else window.location.href = result.payment_url
      }
    } catch (err) { message.error(err.message || '支付激活失败') }
    finally { setActivating(false) }
  }

  // ---- States ----
  if (loading) return <FullPage><div style={{ padding: 80, textAlign: 'center' }}><Spin size="large" /></div></FullPage>
  if (error) return <FullPage><Status icon={<CloseCircleFilled style={{ fontSize: 44, color: '#df1b41' }} />} title="页面加载失败" subtitle={error} /></FullPage>
  if (paid) return <FullPage><Status icon={<CheckCircleFilled style={{ fontSize: 44, color: '#04d66f' }} />} title="支付成功" subtitle="即将跳转..." /></FullPage>
  if (session.status === 'expired') { postToParent({ type: 'hydra-pay:expired', sessionId }); return <FullPage><Status icon={<CloseCircleFilled style={{ fontSize: 44, color: '#bbb' }} />} title="该支付链接已过期" subtitle="请联系商户重新发起支付" /></FullPage> }
  if (session.status === 'completed') { postToParent({ type: 'hydra-pay:completed', sessionId }); return <FullPage><Status icon={<CheckCircleFilled style={{ fontSize: 44, color: '#04d66f' }} />} title="该订单已支付完成" subtitle="如需帮助，请联系商户" /></FullPage> }

  const merchantName = session.merchant_name || '商户'
  const backUrl = session.cancel_url || '#'
  const activeChannel = selectedChannel

  /* =====================================================================
     MAIN CHECKOUT — 50/50 split (like subscription screenshot)
     ===================================================================== */
  return (
    <div style={css.fullPage}>

      {/* ====== LEFT PANEL — Black background ====== */}
      <div style={css.leftPanel}>
        <div style={css.leftContent}>

          {/* Back link */}
          <div style={{ marginBottom: 40 }}>
            {embed ? (
              <button onClick={() => postToParent({ type: 'hydra-pay:cancel', sessionId })} style={css.backLink}>
                <LeftOutlined style={{ fontSize: 12 }} /> 返回
              </button>
            ) : (
              <a href={backUrl} style={{ ...css.backLink, textDecoration: 'none' }}>
                <LeftOutlined style={{ fontSize: 12 }} /> 返回 {merchantName}
              </a>
            )}
          </div>

          {/* Brand / Logo area */}
          <div style={css.brandBlock}>
            <div style={{
              width: 44, height: 44, borderRadius: 10,
              background: '#de481b',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              marginBottom: 20,
            }}>
              <span style={{ color: '#fff', fontSize: 18, fontWeight: 700 }}>星</span>
            </div>
            <h2 style={css.leftTitle}>{merchantName}</h2>
          </div>

          {/* Product info */}
          {session.description && (
            <p style={css.leftDesc}>{session.description}</p>
          )}

          {/* Plan / pricing details — Stripe-style */}
          <div style={{ marginTop: 40 }}>
            <div style={css.planRow}>
              <span style={css.planLabel}>金额</span>
              <span style={css.planValue}>
                ¥ {formatAmount(session.amount)}
              </span>
            </div>
            <div style={css.planDivider} />
            <div style={css.planRow}>
              <span style={css.planLabel}>支付方式</span>
              <span style={css.planValue}>
                {activeChannel === 'wechat' ? '微信支付' : activeChannel === 'unionpay' ? '云闪付' : activeChannel === 'alipay' ? '支付宝' : '—'}
              </span>
            </div>
            {session.expires_at && (
              <>
                <div style={css.planDivider} />
                <div style={css.planRow}>
                  <span style={css.planLabel}>有效期</span>
                  <span style={css.planValue}>
                    <CountdownTimer expiresAt={session.expires_at} inline />
                  </span>
                </div>
              </>
            )}
          </div>

          {/* Total — follows plan details naturally */}
          <div style={{ paddingTop: 28 }}>
            <div style={css.planDivider} />
            <div style={{ ...css.planRow, marginTop: 16 }}>
              <span style={{ ...css.planLabel, fontSize: 14, color: '#a0a0a0' }}>合计</span>
              <span style={{ ...css.planValue, fontSize: 26, fontWeight: 700, color: '#fff' }}>
                ¥ {formatAmount(session.amount)}
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* ====== RIGHT PANEL — White background ====== */}
      <div style={css.rightPanel}>
        <div style={css.rightContent}>

          {/* Green trust badge — Stripe signature */}
          <div style={css.trustBadge}>
            <LockOutlined style={{ fontSize: 12, color: '#04d66f' }} />
            <span>安全加密支付</span>
          </div>

          {/* Heading */}
          <h1 style={css.rightHeading}>支付 {merchantName}</h1>
          {session.description && (
            <p style={css.rightSubheading}>{session.description}</p>
          )}

          {/* ---- Payment method section ---- */}
          <div style={{ ...css.section, marginTop: 32 }}>
            <div style={css.sectionLabel}>支付方式</div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {/* Alipay */}
              <label
                onClick={() => setSelectedChannel('alipay')}
                style={{
                  ...css.methodCard,
                  borderColor: selectedChannel === 'alipay' ? '#1a1a1a' : '#e6e6e6',
                  background: selectedChannel === 'alipay' ? '#fafafa' : '#fff',
                }}>
                <div style={css.methodLogo}>
                  <AlipayLogo height={24} />
                </div>
                <div style={css.methodInfo}>
                  <div style={css.methodName}>支付宝</div>
                  <div style={css.methodHint}>支付宝用户请选择此项</div>
                </div>
                <div style={{
                  ...css.radio,
                  borderColor: selectedChannel === 'alipay' ? '#1a1a1a' : '#d0d0d0',
                }}>
                  {selectedChannel === 'alipay' && <div style={css.radioDot} />}
                </div>
              </label>

              {/* WeChat */}
              <label
                onClick={() => setSelectedChannel('wechat')}
                style={{
                  ...css.methodCard,
                  borderColor: selectedChannel === 'wechat' ? '#1a1a1a' : '#e6e6e6',
                  background: selectedChannel === 'wechat' ? '#fafafa' : '#fff',
                }}>
                <div style={css.methodLogo}>
                  <WechatPayLogo height={18} />
                </div>
                <div style={css.methodInfo}>
                  <div style={css.methodName}>微信支付</div>
                  <div style={css.methodHint}>微信用户请选择此项</div>
                </div>
                <div style={{
                  ...css.radio,
                  borderColor: selectedChannel === 'wechat' ? '#1a1a1a' : '#d0d0d0',
                }}>
                  {selectedChannel === 'wechat' && <div style={css.radioDot} />}
                </div>
              </label>

              {/* UnionPay 云闪付 */}
              <label
                onClick={() => setSelectedChannel('unionpay')}
                style={{
                  ...css.methodCard,
                  borderColor: selectedChannel === 'unionpay' ? '#1a1a1a' : '#e6e6e6',
                  background: selectedChannel === 'unionpay' ? '#fafafa' : '#fff',
                }}>
                <div style={css.methodLogo}>
                  <UnionpayLogo height={26} />
                </div>
                <div style={css.methodInfo}>
                  <div style={css.methodName}>云闪付</div>
                  <div style={css.methodHint}>银联云闪付用户请选择此项</div>
                </div>
                <div style={{
                  ...css.radio,
                  borderColor: selectedChannel === 'unionpay' ? '#1a1a1a' : '#d0d0d0',
                }}>
                  {selectedChannel === 'unionpay' && <div style={css.radioDot} />}
                </div>
              </label>
            </div>
          </div>

          {/* ---- Pay button — Stripe orange #f24b07 ---- */}
          <button
            disabled={!selectedChannel || activating || !!qrResult}
            onClick={handlePay}
            style={{
              ...css.payBtn,
              background: (!selectedChannel || !!qrResult || activating) ? '#e8e8e8' : '#f24b07',
              color: (!selectedChannel || !!qrResult || activating) ? '#bbb' : '#fff',
              cursor: (!selectedChannel || !!qrResult || activating) ? 'not-allowed' : 'pointer',
            }}>
            {activating ? (
              <span style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8 }}>
                <Spin size="small" style={{ color: '#fff' }} /> 处理中...
              </span>
            ) : qrResult ? (
              `扫码支付中 · ¥${formatAmount(session.amount)}`
            ) : (
              `支付 ¥${formatAmount(session.amount)}`
            )}
          </button>

          {/* Terms & Footer — Stripe style, centered below button */}
          <p style={css.terms}>
            点击支付即表示您同意{' '}
            <a href="#" style={css.termsLink}>服务条款</a>
            {' '}和{' '}
            <a href="#" style={css.termsLink}>隐私政策</a>
          </p>

          <p style={css.poweredBy}>
            <LockOutlined style={{ fontSize: 10, color: '#bbb', marginRight: 4 }} />
            <span>Powered by</span>
            <span style={{ fontWeight: 600, marginLeft: 3 }}>HydraPay</span>
          </p>

          {/* ---- QR area after activation ---- */}
          {qrResult && (
            <div style={{ marginTop: 28 }}>
              <div style={css.divider} />
              <div style={{ textAlign: 'center', marginTop: 24 }}>
                <div style={{ marginBottom: 8 }}>
                  {activeChannel === 'wechat' ? <WechatPayLogo height={22} /> : activeChannel === 'unionpay' ? <UnionpayLogo height={28} /> : <AlipayLogo height={32} />}
                </div>
                <p style={{ fontSize: 12, color: '#6b6b6b', margin: '0 0 4px' }}>扫一扫付款</p>
                <p style={{ fontSize: 18, fontWeight: 700, color: '#1a1a1a', margin: '0 0 14px' }}>
                  ¥ {formatAmount(session.amount)}
                </p>
                <QRCodeSection qrCodeURL={qrResult.qr_code_url} paymentURL={qrResult.payment_url} />
                <p style={{ fontSize: 12, color: '#999', margin: '12px 0 0' }}>
                  请使用{activeChannel === 'wechat' ? '微信' : activeChannel === 'unionpay' ? '云闪付' : '支付宝'}扫描二维码支付
                </p>
                <div style={{ marginTop: 8 }}>
                  {session.expires_at && <CountdownTimer expiresAt={session.expires_at} inline />}
                </div>
              </div>
            </div>
          )}
          </div>
      </div>

      {/* ---- Mobile QR Modal ---- */}
      {mobile && (
        <Modal open={!!qrResult && !paid} onCancel={() => setQrResult(null)} footer={null} width={400} centered closable>
          <div style={{ padding: '24px', textAlign: 'center' }}>
            <div style={{ marginBottom: 8 }}>
              {activeChannel === 'wechat' ? <WechatPayLogo height={28} /> : activeChannel === 'unionpay' ? <UnionpayLogo height={36} /> : <AlipayLogo height={40} />}
            </div>
            <p style={{ fontSize: 12, color: '#6b6b6b', margin: '0 0 4px' }}>扫一扫付款</p>
            <p style={{ fontSize: 20, fontWeight: 700, color: '#1a1a1a', margin: '0 0 16px' }}>¥ {formatAmount(session.amount)}</p>
            <QRCodeSection qrCodeURL={qrResult?.qr_code_url} paymentURL={qrResult?.payment_url} />
            <p style={{ fontSize: 12, color: '#999', margin: '12px 0 0' }}>
              请使用{activeChannel === 'wechat' ? '微信' : activeChannel === 'unionpay' ? '云闪付' : '支付宝'}扫描二维码支付
            </p>
            <div style={{ marginTop: 8 }}>{session.expires_at && <CountdownTimer expiresAt={session.expires_at} inline />}</div>
          </div>
        </Modal>
      )}
    </div>
  )
}

// ---- Shared sub-components ----
function FullPage({ children }) {
  return <div style={{ minHeight: '100vh', background: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>{children}</div>
}
function Status({ icon, title, subtitle }) {
  return (
    <div style={{ textAlign: 'center', padding: 60, maxWidth: 420 }}>
      <div style={{ marginBottom: 16 }}>{icon}</div>
      <h3 style={{ fontSize: 18, fontWeight: 600, color: '#1a1a1a', margin: '0 0 8px' }}>{title}</h3>
      <p style={{ color: '#6b6b6b', fontSize: 14, margin: 0 }}>{subtitle}</p>
    </div>
  )
}

// ====== STYLES ======
const css = {
  // Full page — true 50/50 split
  fullPage: {
    display: 'flex',
    minHeight: '100vh',
    fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
  },

  // ---- LEFT PANEL: Black (#000000) ----
  leftPanel: {
    flex: '0 0 50%',
    background: '#000000',
    display: 'flex',
    justifyContent: 'flex-end',
  },
  leftContent: {
    width: '100%',
    maxWidth: 540,
    padding: '48px 56px 56px 56px',
    display: 'flex',
    flexDirection: 'column',
    minHeight: '100vh',
  },
  backLink: {
    display: 'inline-flex', alignItems: 'center', gap: 4,
    fontSize: 13, color: '#a0a0a0', background: 'none', border: 'none',
    cursor: 'pointer', padding: 0,
  },
  brandBlock: {
    marginBottom: 12,
  },
  leftTitle: {
    fontSize: 22,
    fontWeight: 600,
    color: '#ffffff',
    margin: '0 0 8px',
    letterSpacing: '-0.3px',
  },
  leftDesc: {
    fontSize: 14,
    color: '#a0a0a0',
    margin: '0 0 0 0',
    lineHeight: 1.6,
  },

  // Plan details — like subscription checkout
  planRow: {
    display: 'flex', justifyContent: 'space-between', alignItems: 'center',
    padding: '8px 0',
  },
  planLabel: {
    fontSize: 13, color: '#7f7f7f', fontWeight: 450,
  },
  planValue: {
    fontSize: 14, color: '#ffffff', fontWeight: 500,
  },
  planDivider: {
    height: 1, background: 'rgba(255,255,255,0.08)', margin: '8px 0',
  },

  // ---- RIGHT PANEL: White (#ffffff) ----
  rightPanel: {
    flex: '0 0 50%',
    background: '#ffffff',
    display: 'flex',
    justifyContent: 'flex-start',
    overflow: 'auto',
  },
  rightContent: {
    width: '100%',
    maxWidth: 540,
    padding: '56px 56px 56px 56px',
  },

  // Trust badge
  trustBadge: {
    display: 'inline-flex', alignItems: 'center', gap: 5,
    fontSize: 12, fontWeight: 500, color: '#04d66f',
    background: '#eafdf4', padding: '5px 10px', borderRadius: 5,
    marginBottom: 20,
  },

  // Headings
  rightHeading: {
    fontSize: 22, fontWeight: 600, color: '#1a1a1a',
    margin: '0 0 6px', letterSpacing: '-0.3px',
  },
  rightSubheading: {
    fontSize: 14, color: '#6b6b6b', margin: 0, lineHeight: 1.5,
  },

  // Section
  section: { marginBottom: 24 },
  sectionLabel: {
    fontSize: 12, fontWeight: 600, color: '#6b6b6b',
    textTransform: 'uppercase', letterSpacing: '0.5px', marginBottom: 10,
  },

  // Payment method card
  methodCard: {
    display: 'flex', alignItems: 'center', gap: 10,
    padding: '10px 14px', borderRadius: 8,
    border: '1.5px solid #e0e0e0',
    background: '#fff', cursor: 'pointer',
    transition: 'all 0.15s ease', userSelect: 'none',
  },
  methodLogo: {
    width: 94, height: 36, borderRadius: 8,
    background: '#fff', border: '1px solid #eee',
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    flexShrink: 0, padding: '0 6px',
  },
  methodInfo: { flex: 1, minWidth: 0 },
  methodName: { fontSize: 13, fontWeight: 600, color: '#1a1a1a', lineHeight: 1.3 },
  methodHint: { fontSize: 11, color: '#999', marginTop: 1 },

  // Radio
  radio: {
    width: 18, height: 18, borderRadius: '50%',
    border: '2px solid #d0d0d0',
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    flexShrink: 0, transition: 'all 0.15s ease',
  },
  radioDot: {
    width: 8, height: 8, borderRadius: '50%', background: '#1a1a1a',
  },

  // Pay button — exact Stripe orange #f24b07
  payBtn: {
    width: '100%', height: 50, border: 'none', borderRadius: 8,
    fontSize: 16, fontWeight: 600,
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    marginTop: 28,
    transition: 'background 0.15s ease',
  },

  // Terms — centered below pay button
  terms: {
    textAlign: 'center',
    fontSize: 12,
    color: '#999',
    marginTop: 24,
    marginBottom: 0,
    lineHeight: 1.6,
  },
  termsLink: {
    color: '#999',
    textDecoration: 'underline',
    cursor: 'pointer',
  },

  // Powered by — centered
  poweredBy: {
    textAlign: 'center',
    fontSize: 12,
    color: '#bbb',
    marginTop: 16,
    marginBottom: 0,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  },

  // Divider
  divider: { height: 1, background: '#f0f0f0' },
}
