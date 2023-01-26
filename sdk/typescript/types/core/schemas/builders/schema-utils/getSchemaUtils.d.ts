import { BaseSchema, Schema } from "../../Schema";
import { OptionalSchema } from "./types";
export interface SchemaUtils<Raw, Parsed> {
    optional: () => OptionalSchema<Raw, Parsed>;
    transform: <PostTransform>(transformer: BaseSchema<Parsed, PostTransform>) => Schema<Raw, PostTransform>;
}
export declare function getSchemaUtils<Raw, Parsed>(schema: BaseSchema<Raw, Parsed>): SchemaUtils<Raw, Parsed>;
/**
 * schema utils are defined in one file to resolve issues with circular imports
 */
export declare function optional<Raw, Parsed>(schema: BaseSchema<Raw, Parsed>): OptionalSchema<Raw, Parsed>;
export declare function transform<PreTransformRaw, PreTransformParsed, PostTransform>(schema: BaseSchema<PreTransformRaw, PreTransformParsed>, transformer: BaseSchema<PreTransformParsed, PostTransform>): Schema<PreTransformRaw, PostTransform>;
