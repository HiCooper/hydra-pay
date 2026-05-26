package wechat

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/wechatpay-apiv3/wechatpay-go/core"

	"github.com/hydra/pay-service/internal/channel"
	"github.com/hydra/pay-service/internal/model"
	"github.com/hydra/pay-service/pkg/errors"
)

// ---- WeChat Pay OnboardingProvider implementation ----

const (
	applymentSubmitPath = "/v3/applyment4sub/applyment/"
	applymentQueryPath  = "/v3/applyment4sub/applyment/applyment_id/%s"
	applymentSignPath   = "/v3/apply4sub/sub_merchants/%s/application/%s/sign"
)

type wechatApplymentReq struct {
	BusinessCode       string                 `json:"business_code"`
	OutRequestNo       string                 `json:"out_request_no"`
	OrganizationType   string                 `json:"organization_type"`
	ContactInfo        *wechatContactInfo     `json:"contact_info"`
	SubjectInfo        *wechatSubjectInfo     `json:"subject_info"`
	BusinessInfo       *wechatBusinessInfo    `json:"business_info"`
	SettlementInfo     *wechatSettlementInfo  `json:"settlement_info"`
	NotifyURL          string                 `json:"notify_url,omitempty"`
}

type wechatContactInfo struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Email string `json:"email,omitempty"`
}

type wechatSubjectInfo struct {
	SubjectType string `json:"subject_type"`
	Name        string `json:"name"`
}

type wechatBusinessInfo struct {
	MerchantShortname string `json:"merchant_shortname"`
}

type wechatSettlementInfo struct {
	SettlementID string `json:"settlement_id"`
	QualifyType  string `json:"qualify_type"`
}

type wechatApplymentRsp struct {
	ApplymentID string `json:"applyment_id"`
	OutRequestNo string `json:"out_request_no"`
}

type wechatApplymentQueryRsp struct {
	ApplymentID    string `json:"applyment_id"`
	OutRequestNo   string `json:"out_request_no"`
	ApplymentState string `json:"applyment_state"`
	SubMchid       string `json:"sub_mchid,omitempty"`
	SignURL        string `json:"sign_url,omitempty"`
	RejectReason   string `json:"reject_reason,omitempty"`
}

type wechatSignRsp struct {
	SignURL string `json:"sign_url"`
}

func (a *Adapter) SubmitOnboarding(ctx context.Context, req *channel.OnboardingRequest) (*channel.OnboardingResponse, error) {
	body := wechatApplymentReq{
		BusinessCode:     req.OutRequestNo,
		OutRequestNo:     req.OutRequestNo,
		OrganizationType: "SUBJECT_TYPE_ENTERPRISE",
		ContactInfo: &wechatContactInfo{
			Name:  req.ContactName,
			Phone: req.ContactPhone,
			Email: req.ContactEmail,
		},
		SubjectInfo: &wechatSubjectInfo{
			SubjectType: "SUBJECT_TYPE_ENTERPRISE",
			Name:        req.MerchantName,
		},
		BusinessInfo: &wechatBusinessInfo{
			MerchantShortname: truncate(req.MerchantName, 10),
		},
		SettlementInfo: &wechatSettlementInfo{
			SettlementID: "01",
			QualifyType:  "01",
		},
		NotifyURL: req.NotifyURL,
	}

	var rsp wechatApplymentRsp
	result, err := a.client.Post(ctx, applymentSubmitPath, body)
	if err != nil {
		return nil, errors.Wrap(errors.ChannelError, "wechat onboarding submit failed", err)
	}
	if err := core.UnMarshalResponse(result.Response, &rsp); err != nil {
		return nil, errors.Wrap(errors.ChannelError, "wechat onboarding parse response failed", err)
	}

	log.Printf("[wechat] onboarding submitted: out_request_no=%s, applyment_id=%s", req.OutRequestNo, rsp.ApplymentID)

	return &channel.OnboardingResponse{
		ApplymentID: rsp.ApplymentID,
		Status:      model.OnboardingStatusSubmitted,
		RawResponse: structToMap(rsp),
	}, nil
}

func (a *Adapter) QueryOnboarding(ctx context.Context, applymentID string) (*channel.OnboardingStatusResponse, error) {
	path := fmt.Sprintf(applymentQueryPath, applymentID)

	result, err := a.client.Get(ctx, path)
	if err != nil {
		return nil, errors.Wrap(errors.ChannelError, "wechat onboarding query failed", err)
	}

	var rsp wechatApplymentQueryRsp
	if err := core.UnMarshalResponse(result.Response, &rsp); err != nil {
		return nil, errors.Wrap(errors.ChannelError, "wechat onboarding query parse failed", err)
	}

	log.Printf("[wechat] onboarding query: applyment_id=%s, state=%s, sub_mchid=%s", applymentID, rsp.ApplymentState, rsp.SubMchid)

	status := mapWechatApplyState(rsp.ApplymentState)
	statusRsp := &channel.OnboardingStatusResponse{
		ApplymentID:   rsp.ApplymentID,
		Status:        status,
		SubMerchantID: rsp.SubMchid,
		RawResponse:   structToMap(rsp),
	}

	// If approved with sub_mchid, try to get the sign URL
	if status == model.OnboardingStatusApproved && rsp.SubMchid != "" {
		signPath := fmt.Sprintf(applymentSignPath, rsp.SubMchid, applymentID)
		signResult, err := a.client.Get(ctx, signPath)
		if err != nil {
			log.Printf("[wechat] failed to get sign URL: %v", err)
		} else {
			var signRsp wechatSignRsp
			if err := core.UnMarshalResponse(signResult.Response, &signRsp); err == nil {
				statusRsp.SignURL = signRsp.SignURL
			}
		}
	}

	if rsp.SignURL != "" {
		statusRsp.SignURL = rsp.SignURL
	}

	return statusRsp, nil
}

func (a *Adapter) VerifyOnboardingCallback(ctx context.Context, data *channel.CallbackData) (*channel.OnboardingCallbackResult, error) {
	headers := data.Headers
	timestamp := headers["Wechatpay-Timestamp"]
	nonce := headers["Wechatpay-Nonce"]
	signature := headers["Wechatpay-Signature"]
	serial := headers["Wechatpay-Serial"]

	if timestamp == "" || nonce == "" || signature == "" || serial == "" {
		return nil, errors.New(errors.InvalidSignature, "missing wechat onboarding callback headers")
	}

	if err := a.verifySignature(ctx, serial, timestamp, nonce, data.RawBody, signature); err != nil {
		return nil, errors.Wrap(errors.InvalidSignature, "wechat onboarding signature verification failed", err)
	}

	var notification struct {
		ID           string `json:"id"`
		EventType    string `json:"event_type"`
		ResourceType string `json:"resource_type"`
		Resource     struct {
			Algorithm      string `json:"algorithm"`
			Ciphertext     string `json:"ciphertext"`
			AssociatedData string `json:"associated_data"`
			Nonce          string `json:"nonce"`
			OriginalType   string `json:"original_type"`
		} `json:"resource"`
	}
	if err := json.Unmarshal(data.RawBody, &notification); err != nil {
		return nil, errors.New(errors.ValidationError, "failed to parse wechat onboarding callback json")
	}

	plaintext, err := a.decryptResource(
		notification.Resource.Ciphertext,
		notification.Resource.Nonce,
		notification.Resource.AssociatedData,
	)
	if err != nil {
		return nil, errors.Wrap(errors.InvalidSignature, "wechat onboarding callback decryption failed", err)
	}

	var event struct {
		ApplymentID  string `json:"applyment_id"`
		OutRequestNo string `json:"out_request_no"`
		State        string `json:"applyment_state"`
		SubMchid     string `json:"sub_mchid"`
		RejectReason string `json:"reject_reason"`
	}
	if err := json.Unmarshal(plaintext, &event); err != nil {
		return nil, errors.New(errors.ValidationError, "failed to parse wechat onboarding event")
	}

	log.Printf("[wechat] onboarding callback: applyment_id=%s, state=%s, sub_mchid=%s", event.ApplymentID, event.State, event.SubMchid)

	return &channel.OnboardingCallbackResult{
		ApplymentID:   event.ApplymentID,
		OutRequestNo:  event.OutRequestNo,
		Status:        mapWechatApplyState(event.State),
		SubMerchantID: event.SubMchid,
		RejectReason:  event.RejectReason,
		RawBody:       string(data.RawBody),
	}, nil
}

func mapWechatApplyState(state string) string {
	switch state {
	case "EDITTING", "AUDITING", "CHECKING":
		return model.OnboardingStatusAuditing
	case "APPROVED", "FINISHED":
		return model.OnboardingStatusApproved
	case "REJECTED", "REVOKED":
		return model.OnboardingStatusRejected
	default:
		return model.OnboardingStatusSubmitted
	}
}
