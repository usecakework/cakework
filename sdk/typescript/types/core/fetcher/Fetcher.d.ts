import { APIResponse } from "./APIResponse";
export interface Fetcher {
    fetch: FetchFunction;
}
export declare type FetchFunction = (args: Fetcher.Args) => Promise<APIResponse<unknown, Fetcher.Error>>;
export declare namespace Fetcher {
    interface Args {
        url: string;
        method: string;
        headers?: Record<string, string | undefined>;
        queryParameters?: URLSearchParams;
        body?: unknown;
        timeoutMs?: number;
    }
    type Error = FailedStatusCodeError | NonJsonError | TimeoutError | UnknownError;
    interface FailedStatusCodeError {
        reason: "status-code";
        statusCode: number;
        rawBody: string | undefined;
        body: unknown;
    }
    interface NonJsonError {
        reason: "non-json";
        statusCode: number;
        rawBody: string;
    }
    interface TimeoutError {
        reason: "timeout";
    }
    interface UnknownError {
        reason: "unknown";
        errorMessage: string;
    }
}
export declare const fetcher: FetchFunction;
