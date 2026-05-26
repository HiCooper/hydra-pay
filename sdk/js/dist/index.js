import { HydraPayError } from "./errors.js";
import { verifySignature, parseEvent } from "./webhook.js";
export { HydraPayError } from "./errors.js";
export { verifySignature, parseEvent } from "./webhook.js";
const DEFAULT_BASE_URL = "https://api.hydrapay.com/v1";
class ResourceClient {
    baseUrl;
    apiKey;
    constructor(baseUrl, apiKey) {
        this.baseUrl = baseUrl;
        this.apiKey = apiKey;
    }
    async request(method, path, body, idempotencyKey) {
        const headers = {
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
        const json = (await res.json());
        if (!json.success) {
            throw HydraPayError.fromResponse(res.status, json.error ?? { code: "UNKNOWN", message: "Unknown error" });
        }
        return json.data;
    }
}
class PaymentsClient {
    rc;
    constructor(rc) {
        this.rc = rc;
    }
    async create(params, idempotencyKey) {
        return this.rc.request("POST", "/payments/create", params, idempotencyKey);
    }
    async get(id) {
        return this.rc.request("GET", `/payments/${id}`);
    }
}
class CheckoutClient {
    rc;
    constructor(rc) {
        this.rc = rc;
    }
    async create(params, idempotencyKey) {
        return this.rc.request("POST", "/checkout/sessions", params, idempotencyKey);
    }
}
class RefundsClient {
    rc;
    constructor(rc) {
        this.rc = rc;
    }
    async create(params, idempotencyKey) {
        return this.rc.request("POST", "/refunds", params, idempotencyKey);
    }
    async get(id) {
        return this.rc.request("GET", `/refunds/${id}`);
    }
    async list(paymentId) {
        return this.rc.request("GET", `/payments/${paymentId}/refunds`);
    }
}
class SubscriptionsClient {
    rc;
    constructor(rc) {
        this.rc = rc;
    }
    async create(params, idempotencyKey) {
        return this.rc.request("POST", "/subscriptions", params, idempotencyKey);
    }
    async get(id) {
        return this.rc.request("GET", `/subscriptions/${id}`);
    }
    async list(params) {
        const query = new URLSearchParams();
        if (params?.user_id)
            query.set("user_id", params.user_id);
        if (params?.page)
            query.set("page", String(params.page));
        if (params?.page_size)
            query.set("page_size", String(params.page_size));
        let path = "/subscriptions";
        const qs = query.toString();
        if (qs)
            path += "?" + qs;
        return this.rc.request("GET", path);
    }
    async cancel(id, idempotencyKey) {
        return this.rc.request("POST", `/subscriptions/${id}/cancel`, undefined, idempotencyKey);
    }
}
export class HydraPay {
    payments;
    checkout;
    refunds;
    subscriptions;
    webhooks;
    constructor(apiKey, options) {
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
//# sourceMappingURL=index.js.map