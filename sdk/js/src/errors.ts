import type { HydraPayErrorResponse } from "./types.js";

/** Standard error codes returned by the HydraPay API. */
export const ErrorCode = {
  Validation:       "VALIDATION_ERROR",
  NotFound:         "NOT_FOUND",
  Internal:         "INTERNAL_ERROR",
  Unauthorized:     "UNAUTHORIZED",
  PaymentFailed:    "PAYMENT_FAILED",
  ChannelError:     "CHANNEL_ERROR",
  DuplicatePayment: "DUPLICATE_PAYMENT",
  InvalidSignature: "INVALID_SIGNATURE",
} as const;

export type ErrorCode = (typeof ErrorCode)[keyof typeof ErrorCode];

export class HydraPayError extends Error {
  readonly code: string;
  readonly status: number;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "HydraPayError";
    this.code = code;
    this.status = status;
  }

  static fromResponse(status: number, body: HydraPayErrorResponse): HydraPayError {
    return new HydraPayError(status, body.code, body.message);
  }
}
