export class HydraPayError extends Error {
    code;
    status;
    constructor(status, code, message) {
        super(message);
        this.name = "HydraPayError";
        this.code = code;
        this.status = status;
    }
    static fromResponse(status, body) {
        return new HydraPayError(status, body.code, body.message);
    }
}
//# sourceMappingURL=errors.js.map