import type { HydraPayErrorResponse } from "./types.js";
export declare class HydraPayError extends Error {
    readonly code: string;
    readonly status: number;
    constructor(status: number, code: string, message: string);
    static fromResponse(status: number, body: HydraPayErrorResponse): HydraPayError;
}
