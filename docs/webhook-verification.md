# Webhook 签名验证

HydraPay 发出的 Webhook 请求携带 `X-HydraPay-Signature` 请求头，用于验证请求确实来自 HydraPay。

## 签名格式

```
X-HydraPay-Signature: t=1716652800,v1=a1b2c3d4e5f67890abcdef1234567890abcdef1234567890abcdef1234567890
```

- `t` — Unix 时间戳（秒），用于防重放攻击
- `v1` — HMAC-SHA256 签名，输入为 `"{timestamp}.{body}"`

## 验证步骤

1. 从 header 中提取 timestamp 和签名
2. 检查 timestamp 与当前时间的差值是否在 **5 分钟** 以内
3. 用你的 `webhook_secret` 计算 `HMAC-SHA256("{timestamp}.{body}")`
4. 使用恒定时间比较（timing-safe comparison）对比计算结果与收到的签名

---

## 代码示例

### Go

```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "strconv"
    "strings"
    "time"
)

func VerifySignature(secret string, body []byte, header string) bool {
    tIdx := strings.Index(header, "t=")
    vIdx := strings.Index(header, ",v1=")
    if tIdx < 0 || vIdx < 0 || vIdx <= tIdx {
        return false
    }

    ts, err := strconv.ParseInt(header[tIdx+2:vIdx], 10, 64)
    if err != nil {
        return false
    }

    sig := header[vIdx+4:]
    if sig == "" {
        return false
    }

    if abs(time.Now().Unix()-ts) > 300 {
        return false
    }

    mac := hmac.New(sha256.New, []byte(secret))
    fmt.Fprintf(mac, "%d.", ts)
    mac.Write(body)
    expected := hex.EncodeToString(mac.Sum(nil))

    return hmac.Equal([]byte(sig), []byte(expected))
}

func abs(n int64) int64 {
    if n < 0 {
        return -n
    }
    return n
}
```

### Python

```python
import hashlib
import hmac
import time

def verify_signature(secret: str, body: bytes, header: str) -> bool:
    if not header:
        return False

    parts = header.split(",v1=")
    if len(parts) != 2 or not parts[0].startswith("t="):
        return False

    ts_str = parts[0][2:]
    sig = parts[1]
    if not ts_str or not sig:
        return False

    try:
        ts = int(ts_str)
    except ValueError:
        return False

    if abs(int(time.time()) - ts) > 300:
        return False

    payload = f"{ts}.".encode() + body
    expected = hmac.new(
        secret.encode(), payload, hashlib.sha256
    ).hexdigest()

    return hmac.compare_digest(sig, expected)
```

### JavaScript (Node.js)

```javascript
const crypto = require('crypto');

function verifySignature(secret, body, header) {
    if (!header) return false;

    const match = header.match(/^t=(\d+),v1=([a-f0-9]+)$/);
    if (!match) return false;

    const ts = parseInt(match[1], 10);
    const sig = match[2];

    if (Math.abs(Math.floor(Date.now() / 1000) - ts) > 300) {
        return false;
    }

    const payload = `${ts}.${body}`;
    const expected = crypto
        .createHmac('sha256', secret)
        .update(payload)
        .digest('hex');

    return crypto.timingSafeEqual(
        Buffer.from(sig, 'hex'),
        Buffer.from(expected, 'hex')
    );
}
```

---

## Webhook 回调示例

```json
POST /your-webhook-endpoint
Content-Type: application/json
X-HydraPay-Signature: t=1716652800,v1=abc123def456...

{
  "event": "payment.success",
  "payment_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "user_123",
  "plan_id": "plan_pro",
  "amount": 29900,
  "currency": "CNY",
  "status": "paid",
  "channel": "alipay"
}
```

---

## 事件类型

| 事件 | 说明 |
|------|------|
| `payment.success` | 支付成功 |
| `payment.failed` | 支付失败 |
| `payment.refunded` | 退款完成 |

---

## 常见问题

**Q: 密钥在哪里获取？**
在 Developer Portal → Settings → Webhook Secret 中设置。也可由平台管理员在 App 配置中下发。

**Q: 如何防止重放攻击？**
HydraPay 在每个签名中包含了 Unix 时间戳，验证时要求时间戳在 5 分钟内。服务端应额外实现基于 `payment_id` 的去重逻辑。

**Q: 签名不匹配怎么办？**
检查：1) webhook_secret 是否正确 2) 是否使用了原始请求体（未经任何加工）3) 服务器时钟是否同步。
