package ecny

// Default gateway URLs. Override with WithBaseURL() or WithAgency().
const (
	DefaultSandboxBaseURL    = "https://sandbox-api.ecny.gov.cn"
	DefaultProductionBaseURL = "https://api.ecny.gov.cn"
)

// Agency identifiers for WithAgency().
const (
	AgencyYinsheng = "yinsheng"  // 银盛支付
	AgencyYibao    = "yibao"     // 易宝支付
	AgencyHuifubao = "huifubao"  // 汇付宝
	AgencyZhongyi  = "zhongyi"   // 中移电商
)

// Trade types mirroring channel.CreatePaymentRequest.TradeType.
const (
	TradeNative = "native" // QR code for user to scan (主扫)
	TradeApp    = "app"    // encrypted order info for app cashier launch (拉起支付)
)

// Sign methods.
const (
	SignMethodSM2 = "SM2"
)
