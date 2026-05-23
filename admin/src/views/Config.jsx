import { useState, useEffect } from 'react'
import { api } from '../api/index.js'

export default function Config() {
  const [cfg, setCfg] = useState(null)
  useEffect(() => { api.config().then(setCfg) }, [])
  if (!cfg) return <div className="flex items-center justify-center h-64" style={{ color: 'var(--color-text-muted)' }}>加载中...</div>

  const ok = <span className="text-xs font-medium" style={{ color: 'var(--color-success)' }}>✓ 已配置</span>
  const no = <span className="text-xs font-medium" style={{ color: 'var(--color-text-muted)' }}>— 未配置</span>

  const alipayFields = [
    ['App ID', cfg.alipay?.app_id || '—'],
    ['环境', cfg.alipay?.sandbox === 'true' ? '沙箱' : '生产'],
    ['ISV 私钥', cfg.alipay?.key_loaded ? ok : no],
    ['支付宝公钥', cfg.alipay?.pub_loaded ? ok : no],
  ]

  const wechatFields = [
    ['商户号', cfg.wechat?.mch_id || '—'],
    ['证书序列号', cfg.wechat?.serial_no || '—'],
    ['服务商私钥', cfg.wechat?.key_loaded ? ok : no],
  ]

  return (
    <div>
      <h2 className="text-lg font-bold mb-5" style={{ color: 'var(--color-text-primary)' }}>渠道配置</h2>

      <div className="card mb-5">
        <div className="px-5 py-4 border-b" style={{ borderColor: 'var(--color-border)' }}>
          <h3 className="text-sm font-semibold">支付宝 ISV</h3>
        </div>
        <div className="p-5">
          <div className="grid grid-cols-2 gap-y-3 gap-x-8">
            {alipayFields.map(([label, value], i) => (
              <div key={i} className="flex flex-col gap-0.5">
                <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>{label}</span>
                <span className="text-sm">{value}</span>
              </div>
            ))}
            <div className="flex flex-col gap-0.5">
              <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>异步通知地址</span>
              <code className="text-xs" style={{ color: cfg.alipay?.notify_url ? 'var(--color-text-secondary)' : 'var(--color-text-muted)' }}>{cfg.alipay?.notify_url || '— 未配置'}</code>
            </div>
            <div className="flex flex-col gap-0.5">
              <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>同步跳转地址</span>
              <code className="text-xs" style={{ color: cfg.alipay?.return_url ? 'var(--color-text-secondary)' : 'var(--color-text-muted)' }}>{cfg.alipay?.return_url || '— 未配置'}</code>
            </div>
          </div>
        </div>
      </div>

      <div className="card mb-5">
        <div className="px-5 py-4 border-b" style={{ borderColor: 'var(--color-border)' }}>
          <h3 className="text-sm font-semibold">微信支付服务商</h3>
        </div>
        <div className="p-5">
          <div className="grid grid-cols-2 gap-y-3 gap-x-8">
            {wechatFields.map(([label, value], i) => (
              <div key={i} className="flex flex-col gap-0.5">
                <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>{label}</span>
                <span className="text-sm">{value}</span>
              </div>
            ))}
            <div className="flex flex-col gap-0.5">
              <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>异步通知地址</span>
              <code className="text-xs" style={{ color: cfg.wechat?.notify_url ? 'var(--color-text-secondary)' : 'var(--color-text-muted)' }}>{cfg.wechat?.notify_url || '— 未配置'}</code>
            </div>
          </div>
        </div>
      </div>

      <div className="card">
        <div className="px-5 py-4 border-b" style={{ borderColor: 'var(--color-border)' }}>
          <h3 className="text-sm font-semibold">全局 Webhook</h3>
        </div>
        <div className="p-5">
          <div className="flex flex-col gap-0.5">
            <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>兜底 Webhook 地址（应用未单独配置时使用）</span>
            <code className="text-xs" style={{ color: cfg.global_webhook ? 'var(--color-text-secondary)' : 'var(--color-text-muted)' }}>{cfg.global_webhook || '— 未配置'}</code>
          </div>
        </div>
      </div>
    </div>
  )
}
