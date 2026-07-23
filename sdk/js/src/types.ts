// Request / Response types for the HydraPay API.

export interface HydraPayErrorResponse {
  code: string;
  message: string;
}

export interface Pagination {
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
}

// --- Payment ---

export interface CreatePaymentParams {
  user_id: string;
  plan_id?: string;
  amount: number;
  currency?: string;
  channel?: string;
  trade_type?: string;
  success_url?: string;
  cancel_url?: string;
  description?: string;
  open_id?: string;
  channel_app_id?: string;
  client_ip?: string;
  notify_url?: string;
  metadata?: Record<string, unknown>;
}

export interface Payment {
  payment_id: string;
  trade_no: string;
  app_id?: string;
  user_id?: string;
  plan_id?: string;
  amount: number;
  currency: string;
  channel: string;
  status: string;
  payment_url?: string;
  qr_code_url?: string;
  external_id?: string;
  description?: string;
  created_at: string;
  paid_at?: string | null;
}

// --- Checkout Session ---

export interface CreateCheckoutSessionParams {
  amount: number;
  currency?: string;
  description?: string;
  success_url?: string;
  cancel_url?: string;
}

export interface CheckoutSession {
  id: string;
  checkout_url: string;
  expires_at: string;
}

// --- Refund ---

export interface CreateRefundParams {
  trade_no: string;
  refund_amount: number;
  refund_reason?: string;
}

export interface Refund {
  refund_id: string;
  payment_id?: string;
  trade_no: string;
  channel?: string;
  refund_amount: number;
  refund_reason?: string;
  out_request_no?: string;
  status: string;
  channel_refund_id?: string;
  refund_fee?: number;
  error_msg?: string;
  created_at: string;
}

export interface RefundList {
  payment_id: string;
  refunds: Refund[];
}

// --- Subscription ---

export interface CreateSubscriptionParams {
  plan_id: string;
  user_id: string;
}

export interface Subscription {
  id: string;
  plan_id: string;
  plan_name?: string;
  user_id: string;
  status: string;
  current_period_start: string;
  current_period_end: string;
  amount?: number;
  currency?: string;
  interval?: string;
  created_at: string;
}

export interface ListSubscriptionsParams {
  user_id?: string;
  page?: number;
  page_size?: number;
}

// --- Webhook ---

export interface WebhookEvent {
  event: string;
  payment_id: string;
  user_id?: string;
  plan_id?: string;
  amount: number;
  currency?: string;
  status?: string;
  channel?: string;
  refund_amount?: number;
  refund_reason?: string;
  refund_id?: string;
}
