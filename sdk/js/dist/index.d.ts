import { verifySignature, parseEvent } from "./webhook.js";
import type { CreatePaymentParams, Payment, CreateCheckoutSessionParams, CheckoutSession, CreateRefundParams, Refund, RefundList, CreateSubscriptionParams, Subscription, ListSubscriptionsParams } from "./types.js";
export type * from "./types.js";
export { HydraPayError } from "./errors.js";
export { verifySignature, parseEvent } from "./webhook.js";
declare class ResourceClient {
    private readonly baseUrl;
    private readonly apiKey;
    constructor(baseUrl: string, apiKey: string);
    request<T>(method: string, path: string, body?: unknown, idempotencyKey?: string): Promise<T>;
}
declare class PaymentsClient {
    private readonly rc;
    constructor(rc: ResourceClient);
    create(params: CreatePaymentParams, idempotencyKey?: string): Promise<Payment>;
    get(id: string): Promise<Payment>;
}
declare class CheckoutClient {
    private readonly rc;
    constructor(rc: ResourceClient);
    create(params: CreateCheckoutSessionParams, idempotencyKey?: string): Promise<CheckoutSession>;
}
declare class RefundsClient {
    private readonly rc;
    constructor(rc: ResourceClient);
    create(params: CreateRefundParams, idempotencyKey?: string): Promise<Refund>;
    get(id: string): Promise<Refund>;
    list(paymentId: string): Promise<RefundList>;
}
declare class SubscriptionsClient {
    private readonly rc;
    constructor(rc: ResourceClient);
    create(params: CreateSubscriptionParams, idempotencyKey?: string): Promise<Subscription>;
    get(id: string): Promise<Subscription>;
    list(params?: ListSubscriptionsParams): Promise<{
        subscriptions: Subscription[];
    } | Subscription[]>;
    cancel(id: string, idempotencyKey?: string): Promise<{
        status: string;
    }>;
}
export declare class HydraPay {
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
    constructor(apiKey: string, options?: {
        baseUrl?: string;
    });
}
export default HydraPay;
