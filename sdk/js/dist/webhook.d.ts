import type { WebhookEvent } from "./types.js";
/**
 * Verify an X-HydraPay-Signature header against a webhook secret and raw body.
 * Rejects timestamps older than toleranceSeconds (default 300 = 5 minutes).
 */
export declare function verifySignature(payload: string, signatureHeader: string, secret: string, toleranceSeconds?: number): boolean;
/**
 * Parse a webhook payload into a typed WebhookEvent.
 */
export declare function parseEvent(payload: string): WebhookEvent;
