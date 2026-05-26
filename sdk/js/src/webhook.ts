import { createHmac, timingSafeEqual } from "node:crypto";
import type { WebhookEvent } from "./types.js";

const TOLERANCE_SECONDS = 300;

function parseSignatureHeader(header: string): { timestamp: number; sig: string } | null {
  const tIdx = header.indexOf("t=");
  const vIdx = header.indexOf(",v1=");
  if (tIdx < 0 || vIdx < 0 || vIdx <= tIdx) return null;

  const ts = parseInt(header.slice(tIdx + 2, vIdx), 10);
  if (isNaN(ts)) return null;

  const sig = header.slice(vIdx + 4);
  if (!sig) return null;

  return { timestamp: ts, sig };
}

/**
 * Verify an X-HydraPay-Signature header against a webhook secret and raw body.
 * Rejects timestamps older than toleranceSeconds (default 300 = 5 minutes).
 */
export function verifySignature(
  payload: string,
  signatureHeader: string,
  secret: string,
  toleranceSeconds: number = TOLERANCE_SECONDS
): boolean {
  if (!secret || !signatureHeader) return false;

  const parsed = parseSignatureHeader(signatureHeader);
  if (!parsed) return false;

  const now = Math.floor(Date.now() / 1000);
  if (Math.abs(now - parsed.timestamp) > toleranceSeconds) return false;

  const hmac = createHmac("sha256", secret);
  hmac.update(`${parsed.timestamp}.`);
  hmac.update(payload);
  const expected = hmac.digest("hex");

  // Constant-time comparison
  const expectedBuf = Buffer.from(expected);
  const actualBuf = Buffer.from(parsed.sig);
  if (expectedBuf.length !== actualBuf.length) return false;
  return timingSafeEqual(expectedBuf, actualBuf);
}

/**
 * Parse a webhook payload into a typed WebhookEvent.
 */
export function parseEvent(payload: string): WebhookEvent {
  return JSON.parse(payload) as WebhookEvent;
}
