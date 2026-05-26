package alipay

import (
	"context"
	"fmt"
	"net/url"

	"github.com/smartwalle/alipay/v3"

	"github.com/hydra/pay-service/internal/channel"
	"github.com/hydra/pay-service/internal/model"
	"github.com/hydra/pay-service/pkg/errors"
	"github.com/hydra/pay-service/pkg/logger"
)

// ---- Alipay OnboardingProvider implementation ----

const (
	applyMethodCreate = "ant.merchant.expand.indirect.create"
	applyMethodQuery  = "ant.merchant.expand.indirect.query"
	applyVersion      = "1.0"
)

// antMerchantCreateParam implements alipay.Param for ant.merchant.expand.indirect.create
type antMerchantCreateParam struct {
	alipay.AuxParam
	OutBizNo      string `json:"out_biz_no"`
	ExternalID    string `json:"external_id"`
	Name          string `json:"name"`
	AliasName     string `json:"alias_name"`
	ServicePhone  string `json:"service_phone"`
	ContactName   string `json:"contact_name"`
	ContactPhone  string `json:"contact_phone"`
	ContactEmail  string `json:"contact_email"`
	CategoryID    string `json:"category_id"`
	Source        string `json:"source"`
	NotifyURL     string `json:"notify_url,omitempty"`
}

func (p antMerchantCreateParam) APIName() string      { return applyMethodCreate }
func (p antMerchantCreateParam) Params() map[string]string { return nil }

// antMerchantCreateRsp is the response for ant.merchant.expand.indirect.create
type antMerchantCreateRsp struct {
	alipay.Error
	OrderID  string `json:"order_id"`
	SubMerchantID string `json:"sub_merchant_id"`
}

// antMerchantQueryParam implements alipay.Param for ant.merchant.expand.indirect.query
type antMerchantQueryParam struct {
	alipay.AuxParam
	OutBizNo string `json:"out_biz_no,omitempty"`
	OrderID  string `json:"order_id,omitempty"`
}

func (p antMerchantQueryParam) APIName() string      { return applyMethodQuery }
func (p antMerchantQueryParam) Params() map[string]string { return nil }

// antMerchantQueryRsp is the response for ant.merchant.expand.indirect.query
type antMerchantQueryRsp struct {
	alipay.Error
	OrderID       string `json:"order_id"`
	Status        string `json:"status"`
	SubMerchantID string `json:"sub_merchant_id"`
	SignURL       string `json:"sign_url"`
	QrCodeURL     string `json:"qr_code_url"`
	ApplyTime     string `json:"apply_time"`
	AuditTime     string `json:"audit_time"`
	FailReason    string `json:"fail_reason"`
	FailDetail    string `json:"fail_detail"`
}

func (a *Adapter) SubmitOnboarding(ctx context.Context, req *channel.OnboardingRequest) (*channel.OnboardingResponse, error) {
	param := antMerchantCreateParam{
		OutBizNo:     req.OutRequestNo,
		ExternalID:   req.OutRequestNo,
		Name:         req.MerchantName,
		AliasName:    req.MerchantName,
		ServicePhone: req.ContactPhone,
		ContactName:  req.ContactName,
		ContactPhone: req.ContactPhone,
		ContactEmail: req.ContactEmail,
		CategoryID:   "2015091000020000",
		Source:       "H5_PARTNER_APPLY",
	}
	if req.NotifyURL != "" {
		param.NotifyURL = req.NotifyURL
	}

	var rsp antMerchantCreateRsp
	if err := a.client.Request(ctx, param, &rsp); err != nil {
		return nil, errors.Wrap(errors.ChannelError, "alipay onboarding create failed", err)
	}
	if rsp.Code != "10000" {
		return nil, errors.New(errors.ChannelError,
			fmt.Sprintf("alipay onboarding create error: %s (code=%s, sub_code=%s)", rsp.Msg, rsp.Code, rsp.SubCode))
	}

	logger.Info(ctx, "onboarding submitted", "out_biz_no", req.OutRequestNo, "order_id", rsp.OrderID)

	// Query for sign URL
	var queryRsp antMerchantQueryRsp
	queryParam := antMerchantQueryParam{OrderID: rsp.OrderID}
	if err := a.client.Request(ctx, queryParam, &queryRsp); err != nil {
		logger.Error(ctx, "onboarding query for sign URL failed", "error", err)
	}

	return &channel.OnboardingResponse{
		ApplymentID: rsp.OrderID,
		SignURL:     queryRsp.SignURL,
		QRCodeURL:   queryRsp.QrCodeURL,
		Status:      mapAlipayApplyStatus(queryRsp.Status),
		RawResponse: structToMap(rsp),
	}, nil
}

func (a *Adapter) QueryOnboarding(ctx context.Context, applymentID string) (*channel.OnboardingStatusResponse, error) {
	var rsp antMerchantQueryRsp
	param := antMerchantQueryParam{OrderID: applymentID}
	if err := a.client.Request(ctx, param, &rsp); err != nil {
		return nil, errors.Wrap(errors.ChannelError, "alipay onboarding query failed", err)
	}
	if rsp.Code != "10000" {
		return nil, errors.New(errors.ChannelError,
			fmt.Sprintf("alipay onboarding query error: %s (code=%s)", rsp.Msg, rsp.Code))
	}

	return &channel.OnboardingStatusResponse{
		ApplymentID:   rsp.OrderID,
		Status:        mapAlipayApplyStatus(rsp.Status),
		SubMerchantID: rsp.SubMerchantID,
		SignURL:       rsp.SignURL,
		QRCodeURL:     rsp.QrCodeURL,
		RawResponse:   structToMap(rsp),
	}, nil
}

func (a *Adapter) VerifyOnboardingCallback(ctx context.Context, data *channel.CallbackData) (*channel.OnboardingCallbackResult, error) {
	values, err := url.ParseQuery(string(data.RawBody))
	if err != nil {
		return nil, errors.New(errors.InvalidSignature, "failed to parse alipay onboarding callback body")
	}

	if err := a.client.VerifySign(ctx, values); err != nil {
		return nil, errors.Wrap(errors.InvalidSignature, "alipay onboarding callback signature verification failed", err)
	}

	notifyType := values.Get("notify_type")
	orderID := values.Get("order_id")
	resultCode := values.Get("result_code")
	subMerchantID := values.Get("sub_merchant_id")
	rejectReason := values.Get("fail_reason")

	logger.Info(ctx, "onboarding callback", "order_id", orderID, "notify_type", notifyType, "result", resultCode)

	status := model.OnboardingStatusAuditing
	if resultCode == "SUCCESS" {
		status = model.OnboardingStatusApproved
	} else if resultCode == "FAIL" {
		status = model.OnboardingStatusRejected
	}

	return &channel.OnboardingCallbackResult{
		ApplymentID:   orderID,
		Status:        status,
		SubMerchantID: subMerchantID,
		RejectReason:  rejectReason,
		RawBody:       string(data.RawBody),
	}, nil
}

func mapAlipayApplyStatus(status string) string {
	switch status {
	case "CHECKING", "AUDITING":
		return model.OnboardingStatusAuditing
	case "SUCCESS", "FINISH":
		return model.OnboardingStatusApproved
	case "FAIL":
		return model.OnboardingStatusRejected
	default:
		return model.OnboardingStatusSubmitted
	}
}
