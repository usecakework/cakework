export declare type addQuestionMarksToNullableProperties<T> = {
    [K in OptionalKeys<T>]?: undefined extends T[K] ? T[K] : never;
} & Pick<T, RequiredKeys<T>>;
export declare type OptionalKeys<T> = {
    [K in keyof T]: undefined extends T[K] ? K : never;
}[keyof T];
export declare type RequiredKeys<T> = {
    [K in keyof T]: undefined extends T[K] ? never : K;
}[keyof T];
