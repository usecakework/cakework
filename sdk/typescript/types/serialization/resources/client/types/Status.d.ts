/**
 * This file was auto-generated by Fern from our API Definition.
 */
import * as serializers from "../../..";
import { CakeworkApi } from "../../../..";
import * as core from "../../../../core";
export declare const Status: core.serialization.Schema<serializers.Status.Raw, CakeworkApi.Status>;
export declare namespace Status {
    type Raw = "PENDING" | "IN_PROGRESS" | "SUCCEEDED" | "FAILED";
}