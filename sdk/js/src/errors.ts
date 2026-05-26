import type { HydraPayErrorResponse } from "./types.js";

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
