import { Schema } from "../../Schema";
export declare const OPTIONAL_BRAND: {
    _isOptional: void;
};
export declare type OptionalSchema<Raw, Parsed> = Schema<Raw | null | undefined, Parsed | undefined> & {
    _isOptional: void;
};
