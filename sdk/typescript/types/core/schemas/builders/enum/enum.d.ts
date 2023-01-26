import { Schema } from "../../Schema";
export declare function enum_<U extends string, E extends Readonly<[U, ...U[]]>>(_values: E): Schema<E[number], E[number]>;
