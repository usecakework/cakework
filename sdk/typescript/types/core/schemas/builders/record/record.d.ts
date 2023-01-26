import { Schema } from "../../Schema";
export declare function record<RawKey extends string | number, ParsedKey extends string | number, RawValue, ParsedValue>(keySchema: Schema<RawKey, ParsedKey>, valueSchema: Schema<RawValue, ParsedValue>): Schema<Record<RawKey, RawValue>, Record<ParsedKey, ParsedValue>>;
