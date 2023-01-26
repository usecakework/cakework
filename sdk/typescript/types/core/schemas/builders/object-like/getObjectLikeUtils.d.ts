import { BaseObjectLikeSchema, ObjectLikeSchema, ObjectLikeUtils } from "./types";
export declare function getObjectLikeUtils<Raw, Parsed>(schema: BaseObjectLikeSchema<Raw, Parsed>): ObjectLikeUtils<Raw, Parsed>;
/**
 * object-like utils are defined in one file to resolve issues with circular imports
 */
export declare function withProperties<RawObjectShape, ParsedObjectShape, Properties>(objectLike: BaseObjectLikeSchema<RawObjectShape, ParsedObjectShape>, properties: {
    [K in keyof Properties]: Properties[K] | ((parsed: ParsedObjectShape) => Properties[K]);
}): ObjectLikeSchema<RawObjectShape, ParsedObjectShape & Properties>;
