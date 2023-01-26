import { SchemaUtils } from "./builders";
export declare type Schema<Raw = unknown, Parsed = unknown> = BaseSchema<Raw, Parsed> & SchemaUtils<Raw, Parsed>;
export declare type inferRaw<S extends Schema> = S extends Schema<infer Raw, any> ? Raw : never;
export declare type inferParsed<S extends Schema> = S extends Schema<any, infer Parsed> ? Parsed : never;
export interface BaseSchema<Raw, Parsed> {
    parse: (raw: Raw, opts?: SchemaOptions) => Parsed | Promise<Parsed>;
    json: (parsed: Parsed, opts?: SchemaOptions) => Raw | Promise<Raw>;
}
export interface SchemaOptions {
    /**
     * @default false
     */
    skipUnknownKeysOnParse?: boolean;
    /**
     * @default false
     */
    includeUnknownKeysOnJson?: boolean;
}
