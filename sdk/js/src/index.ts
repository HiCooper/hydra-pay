import { HydraPayError } from "./errors.js";
import { verifySignature, parseEvent } from "./webhook.js";
import type {
  CreatePaymentParams,
  Payment,
  CreateCheckoutSessionParams,
  CheckoutSession,
  CreateRefundParams,
  Refund,
  RefundList,
  CreateSubscriptionParams,
  Subscription,
  ListSubscriptionsParams,
  WebhookEvent,
  Pagination,
  HydraPayErrorResponse,
} from "./types.js";

export type * from "./types.js";
export { HydraPayError } from "./errors.js";
export { verifySignature, parseEvent } from "./webhook.js";

const DEFAULT_BASE_URL = "https://api.hydrapay.com/v1";

interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: HydraPayErrorResponse;
  pagination?: Pagination;
}

class ResourceClient {
  constructor(
    private readonly baseUrl: string,
    private readonly apiKey: string
  ) {}

  async request<T>(
    method: string,
    path: string,
    body?: unknown,
    idempotencyKey?: string
  ): Promise<T> {
    const headers: Record<string, string> = {
      "X-API-Key": this.apiKey,
      "Content-Type": "application/json",
      Accept: "application/json",
    };
    if (idempotencyKey) {
      headers["Idempotency-Key"] = idempotencyKey;
    }

    const res = await fetch(this.baseUrl + path, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });

    const json = (await res.json()) as ApiResponse<T>;

    if (!json.success) {
      throw HydraPayError.fromResponse(res.status, json.error ?? { code: "UNKNOWN", message: "Unknown error" });
    }

    return json.data as T;
  }
}

class PaymentsClient {
  constructor(private readonly rc: ResourceClient) {}

  async create(params: CreatePaymentParams, idempotencyKey?: string): Promise<Payment> {
    return this.rc.request<Payment>("POST", "/payments/create", params, idempotencyKey);
  }

  async get(id: string): Promise<Payment> {
    return this.rc.request<Payment>("GET", `/payments/${id}`);
  }
}

class CheckoutClient {
  constructor(private readonly rc: ResourceClient) {}

  async create(params: CreateCheckoutSessionParams, idempotencyKey?: string): Promise<CheckoutSession> {
    return this.rc.request<CheckoutSession>("POST", "/checkout/sessions", params, idempotencyKey);
  }
}

class RefundsClient {
  constructor(private readonly rc: ResourceClient) {}

  async create(params: CreateRefundParams, idempotencyKey?: string): Promise<Refund> {
    return this.rc.request<Refund>("POST", "/refunds", params, idempotencyKey);
  }

  async get(id: string): Promise<Refund> {
    return this.rc.request<Refund>("GET", `/refunds/${id}`);
  }

  async list(paymentId: string): Promise<RefundList> {
    return this.rc.request<RefundList>("GET", `/payments/${paymentId}/refunds`);
  }
}

class SubscriptionsClient {
  constructor(private readonly rc: ResourceClient) {}

  async create(params: CreateSubscriptionParams, idempotencyKey?: string): Promise<Subscription> {
    return this.rc.request<Subscription>("POST", "/subscriptions", params, idempotencyKey);
  }

  async get(id: string): Promise<Subscription> {
    return this.rc.request<Subscription>("GET", `/subscriptions/${id}`);
  }

  async list(params?: ListSubscriptionsParams): Promise<{ subscriptions: Subscription[] } | Subscription[]> {
    const query = new URLSearchParams();
    if (params?.user_id) query.set("user_id", params.user_id);
    if (params?.page) query.set("page", String(params.page));
    if (params?.page_size) query.set("page_size", String(params.page_size));

    let path = "/subscriptions";
    const qs = query.toString();
    if (qs) path += "?" + qs;

    return this.rc.request<{ subscriptions: Subscription[] } | Subscription[]>("GET", path);
  }

  async cancel(id: string, idempotencyKey?: string): Promise<{ status: string }> {
    return this.rc.request<{ status: string }>("POST", `/subscriptions/${id}/cancel`, undefined, idempotencyKey);
  }
}

export class HydraPay {
  readonly payments: PaymentsClient;
  readonly checkout: CheckoutClient;
  readonly refunds: RefundsClient;
  readonly subscriptions: SubscriptionsClient;
  readonly webhooks: {
    /** Verify an X-HydraPay-Signature header. @deprecated Use the top-level `verifySignature` export instead. */
    verifySignature: typeof verifySignature;
    /** Parse a webhook payload into a typed event. @deprecated Use the top-level `parseEvent` export instead. */
    parseEvent: typeof parseEvent;
  };

  constructor(apiKey: string, options?: { baseUrl?: string }) {
    const baseUrl = (options?.baseUrl ?? DEFAULT_BASE_URL).replace(/\/+$/, "");
    const rc = new ResourceClient(baseUrl, apiKey);
    this.payments = new PaymentsClient(rc);
    this.checkout = new CheckoutClient(rc);
    this.refunds = new RefundsClient(rc);
    this.subscriptions = new SubscriptionsClient(rc);
    this.webhooks = { verifySignature, parseEvent };
  }
}

export default HydraPay;
