import { AlipayLogo, WechatPayLogo } from './ChannelLogos'

const channelConfig = {
  alipay: {
    label: '支付宝',
    sublabel: '推荐支付宝用户使用',
    color: '#1677ff',
    bg: '#f0f5ff',
    borderColor: '#1677ff',
    Logo: AlipayLogo,
    logoHeight: 28,
  },
  wechat: {
    label: '微信支付',
    sublabel: '推荐微信用户使用',
    color: '#09BB07',
    bg: '#f0fdf4',
    borderColor: '#09BB07',
    Logo: WechatPayLogo,
    logoHeight: 20,
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
        gap: 12,
        padding: '14px 16px',
        borderRadius: 8,
        border: active ? `2px solid ${cfg.borderColor}` : '2px solid #e8e8e8',
        background: active ? cfg.bg : '#fff',
        cursor: 'pointer',
        transition: 'all 0.2s',
      }}
      onMouseEnter={e => {
        if (!active) e.currentTarget.style.borderColor = cfg.borderColor
      }}
      onMouseLeave={e => {
        if (!active) e.currentTarget.style.borderColor = '#e8e8e8'
      }}
    >
      <Logo height={cfg.logoHeight} />
      <div style={{ flex: 1 }}>
        <div style={{ fontSize: 15, fontWeight: 600, color: '#333' }}>{cfg.label}</div>
        <div style={{ fontSize: 11, color: '#bbb', marginTop: 2 }}>{cfg.sublabel}</div>
      </div>
      {active && (
        <span style={{
          width: 18, height: 18, borderRadius: '50%',
          background: cfg.color, color: '#fff',
          display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
          fontSize: 12, fontWeight: 700,
        }}>
          ✓
        </span>
      )}
    </div>
  )
}
