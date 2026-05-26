import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { createHmac } from "node:crypto";
import { verifySignature, parseEvent } from "../src/webhook.js";

const SECRET = "whsec_test_12345";

function sign(payload: string, secret: string, ts?: number): string {
  const timestamp = ts ?? Math.floor(Date.now() / 1000);
  const hmac = createHmac("sha256", secret);
  hmac.update(`${timestamp}.`);
  hmac.update(payload);
  const sig = hmac.digest("hex");
  return `t=${timestamp},v1=${sig}`;
}

describe("verifySignature", () => {
  it("accepts a valid signature", () => {
    const payload = JSON.stringify({ event: "payment.success", payment_id: "p_001" });
    const header = sign(payload, SECRET);
    assert.equal(verifySignature(payload, header, SECRET), true);
  });

  it("rejects an expired timestamp", () => {
    const payload = JSON.stringify({ event: "payment.success" });
    const old = Math.floor(Date.now() / 1000) - 600;
    const header = sign(payload, SECRET, old);
    assert.equal(verifySignature(payload, header, SECRET), false);
  });

  it("rejects a wrong secret", () => {
    const payload = JSON.stringify({ event: "payment.success" });
    const header = sign(payload, SECRET);
    assert.equal(verifySignature(payload, header, "wrong_secret"), false);
  });

  it("rejects a tampered body", () => {
    const header = sign(JSON.stringify({ event: "payment.success" }), SECRET);
    assert.equal(verifySignature(JSON.stringify({ event: "payment.failed" }), header, SECRET), false);
  });

  it("rejects empty inputs", () => {
    assert.equal(verifySignature("{}", "", "secret"), false);
    assert.equal(verifySignature("{}", "t=1,v1=abc", ""), false);
  });
});

describe("parseEvent", () => {
  it("parses a payment.success event", () => {
    const payload = JSON.stringify({
      event: "payment.success",
      payment_id: "p_001",
      amount: 29900,
      currency: "CNY",
      channel: "alipay",
    });
    const event = parseEvent(payload);
    assert.equal(event.event, "payment.success");
    assert.equal(event.payment_id, "p_001");
    assert.equal(event.amount, 29900);
    assert.equal(event.currency, "CNY");
  });

  it("parses a payment.refunded event with refund fields", () => {
    const payload = JSON.stringify({
      event: "payment.refunded",
      payment_id: "p_001",
      refund_id: "r_001",
      refund_amount: 10000,
      refund_reason: "customer request",
    });
    const event = parseEvent(payload);
    assert.equal(event.event, "payment.refunded");
    assert.equal(event.refund_amount, 10000);
    assert.equal(event.refund_id, "r_001");
  });

  it("throws on invalid JSON", () => {
    assert.throws(() => parseEvent("not json"));
  });
});
