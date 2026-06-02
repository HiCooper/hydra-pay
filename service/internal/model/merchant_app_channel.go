package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// MerchantAppChannel records which payment channel an app has activated.
// Sub-merchant IDs (alipay_pid, wechat_sub_mchid, etc.) live on the Merchant
// model — one per channel per legal entity.  This table only tracks per-app
// enablement and channel-specific config overrides.
type MerchantAppChannel struct {
	ID         uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	MerchantID uuid.UUID      `gorm:"type:uuid;not null;index"                       json:"merchant_id"`
	AppID      uuid.UUID      `gorm:"type:uuid;not null;index:idx_app_channel,unique" json:"app_id"`
	ChannelKey string         `gorm:"type:varchar(32);not null;index:idx_app_channel,unique" json:"channel_key"`
	Enabled    bool           `gorm:"default:false"                                  json:"enabled"`
	Config     datatypes.JSON `gorm:"type:jsonb"                                     json:"config"`
	SortOrder  int            `gorm:"default:0"                                      json:"sort_order"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}
