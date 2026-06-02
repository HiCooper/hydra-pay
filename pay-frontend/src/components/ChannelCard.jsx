import { AlipayLogo, WechatPayLogo, UnionpayLogo } from './ChannelLogos'

const channelConfig = {
  alipay: {
    label: '支付宝',
    sublabel: '推荐支付宝用户使用',
    color: '#1677ff',
    bg: '#f5f9ff',
    borderColor: '#1677ff',
    Logo: AlipayLogo,
    logoHeight: 28,
  },
  wechat: {
    label: '微信支付',
    sublabel: '推荐微信用户使用',
    color: '#07c160',
    bg: '#f5fdf8',
    borderColor: '#07c160',
    Logo: WechatPayLogo,
    logoHeight: 20,
  },
  unionpay: {
    label: '云闪付',
    sublabel: '银联云闪付用户请选择此项',
    color: '#C41230',
    bg: '#fef5f6',
    borderColor: '#C41230',
    Logo: UnionpayLogo,
    logoHeight: 24,
  },
}

export default function ChannelCard({ channel, active, onClick }) {
  const cfg = channelConfig[channel]
  if (!cfg) return null
  const { Logo } = cfg

  return (
    <div
      onClick={onClick}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 14,
        padding: '14px 18px',
        borderRadius: 8,
        border: active ? `1.5px solid ${cfg.borderColor}` : '1.5px solid #e6e6e6',
        background: active ? cfg.bg : '#fff',
        cursor: 'pointer',
        transition: 'all 0.2s ease',
        userSelect: 'none',
      }}
      onMouseEnter={e => {
        if (!active) e.currentTarget.style.borderColor = '#c0c0c0'
        e.currentTarget.style.background = active ? cfg.bg : '#fafafa'
      }}
      onMouseLeave={e => {
        if (!active) e.currentTarget.style.borderColor = '#e6e6e6'
        e.currentTarget.style.background = active ? cfg.bg : '#fff'
      }}
    >
      {/* Logo */}
      <div style={{
        width: 44, height: 44, borderRadius: 8,
        background: '#fff',
        border: '1px solid #f0f0f0',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        flexShrink: 0,
      }}>
        <Logo height={cfg.logoHeight} />
      </div>

      {/* Labels */}
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontSize: 14, fontWeight: 600, color: '#1a1a1a', lineHeight: 1.3 }}>
          {cfg.label}
        </div>
        <div style={{ fontSize: 11, color: '#999', marginTop: 2 }}>
          {cfg.sublabel}
        </div>
      </div>

      {/* Radio indicator */}
      <div style={{
        width: 20, height: 20, borderRadius: '50%',
        border: active ? `2px solid ${cfg.borderColor}` : '2px solid #d0d0d0',
        background: active ? cfg.borderColor : '#fff',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        flexShrink: 0,
        transition: 'all 0.2s ease',
      }}>
        {active && (
          <svg width="10" height="10" viewBox="0 0 10 10" fill="none">
            <path d="M2 5L4 7L8 3" stroke="#fff" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
          </svg>
        )}
      </div>
    </div>
  )
}
