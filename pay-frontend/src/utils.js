export function isMobile() {
  if (typeof navigator === 'undefined') return false
  return /Android|iPhone|iPad|iPod|webOS/i.test(navigator.userAgent) || window.innerWidth < 768
}

export function formatAmount(amount) {
  return (amount / 100).toFixed(2)
}

export function getChannelLabel(channel) {
  switch (channel) {
    case 'alipay': return '支付宝'
    case 'wechat': return '微信支付'
    default: return channel
  }
}
