export function AlipayLogo({ height = 32 }) {
  return (
    <img src={`${import.meta.env.BASE_URL}alipay_logo.svg`} alt="支付宝" style={{ height }} />
  )
}

export function WechatPayLogo({ height = 24 }) {
  return (
    <img src={`${import.meta.env.BASE_URL}wechat_pay_logo.svg`} alt="微信支付" style={{ height }} />
  )
}

export function UnionpayLogo({ height = 24 }) {
  return (
    <img src={`${import.meta.env.BASE_URL}unionpay_logo.svg`} alt="云闪付" style={{ height }} />
  )
}
