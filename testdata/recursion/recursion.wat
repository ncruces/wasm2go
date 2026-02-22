;; Examples of recursive functions.
;;
;; factorial: a recursive function that calls itself.
;; is_even and is_odd: mutually recursive functions that call each other.
;;
;; Eli Bendersky [https://eli.thegreenplace.net]
;; This code is in the public domain.
(module
    ;; factorial(n) returns the factorial of n.
    (func $factorial (export "factorial") (param $n i32) (result i32)
        (if (result i32)
            (i32.le_s (local.get $n) (i32.const 1))
            (then (i32.const 1))
            (else
                (i32.mul
                    (local.get $n)
                    (call $factorial (i32.sub (local.get $n) (i32.const 1)))))
        )
    )

    ;; is_odd(n) returns 1 if n is odd, 0 otherwise.
    (func $is_even (export "is_even") (param $n i32) (result i32)
        (if (result i32)
            (i32.eqz (local.get $n))
            (then (i32.const 1))
            (else (call $is_odd (i32.sub (local.get $n) (i32.const 1))))
        )
    )

    ;; is_even(n) returns 1 if n is even, 0 otherwise.
    (func $is_odd (export "is_odd") (param $n i32) (result i32)
        (if (result i32)
            (i32.eqz (local.get $n))
            (then (i32.const 0))
            (else (call $is_even (i32.sub (local.get $n) (i32.const 1))))
        )
    )
)