;; The WASM spec's https://webassembly.github.io/spec/core/exec/instructions.html
;; page explains how instructions interact with the stack.
;;
;; Stack illustrations in comments follow the convention that the top of the
;; stack is on the left, e.g.:
;;
;;    [x y ...]
;;
;; Means x is on top of the stack, then y below it, then other stuff below that.
;;
;; Eli Bendersky [https://eli.thegreenplace.net]
;; This code is in the public domain.
(module
    ;; sub(a, b) returns a-b
    (func $sub (param $a i32) (param $b i32) (result i32)
        ;; When the function is called, its parameters are automatically mapped
        ;; from the stack to $a and $b

        ;; The sub instruction has two parameters, it's a binop according to
        ;; the WASM spec, working as follows:
        ;;   1. pop c2 from the stack
        ;;   2. pop c1 from the stack
        ;;   3. c = binop(c1, c2)
        ;;   4. push c to stack
        ;;
        ;; So in our case, we have to push $a first, then $b to get the
        ;; stack [b a ...]
        local.get $a
        local.get $b

        ;; After calling sub, the stack will have [(a-b) ...] on it. We don't
        ;; have to do anything else because this is exactly what's expected
        ;; of a function having a single return value.
        i32.sub
    )

    ;; Demonstrates function calls with an explicit stack; retuns a constant
    ;; that's the result of subtracting 91-23.
    (func $stack_func_call (export "stack_func_call") (result i32)
        ;; A function expects its parameters on the stack: the first param
        ;; is the "deepest". E.g. to call sub(91, 23), we need the stack:
        ;;
        ;;   [23 91 ...]
        ;;
        ;; Prior to calling $sub. This makes the order of pushes natural.
        ;;
        ;; This call is equivalent to the folded form:
        ;;  (call $sub (i32.const 91) (i32.const 23))
        i32.const 91
        i32.const 23
        call $sub
    )

    (func $greater (export "greater") (param $a i32) (param $b i32) (result i32)
        ;; Load a and b on the stack and perform a>b
        local.get $a
        local.get $b
        i32.gt_s

        ;; The stack form of 'if':
        ;;   1. pop c from stack
        ;;   2. if c is non-zero then execute the first block
        ;;   3. otherwise execute the else block
        ;;
        ;; Since we don't explicitly "return" from these blocks, we have to
        ;; annotate the 'if' with its return type, otherwise validation will
        ;; complain.
        if (result i32)
            i32.const 1
        else
            i32.const 0
        end

        ;; An alternative way to write this function using the folded form
        ;; would be:
        ;; (if (result i32) (i32.gt_s (local.get $a) (local.get $b))
        ;;     (then (i32.const 1))
        ;;     (else (i32.const 0)))
    )

    ;; Demonstrates the "tee" instruction in action.
    (func $two_a_plus_b (export "two_a_plus_b") (param $a i32) (param $b i32) (result i32)
        (local $tmp i32)
        ;; This example is contrived, could be accomplished more easily
        ;; using other means!
        local.get $b
        local.get $a
        
        ;; Take $a from the stack, copy it to $tmp, but also keep it on the
        ;; stack for the next instruction.
        local.tee $tmp

        ;; Stack: [a b ...] --> [a+b ...]
        i32.add

        ;; Stack [a+b ...] --> [a+b+a ...]
        local.get $tmp
        i32.add
    )

    ;; Demonstrates using local.tee in folded form to produce two values;
    ;; returns a+a+b, a+b
    (func $tee_for_two (export "tee_for_two") (param $a i32) (param $b i32) (result i32 i32)
        (local $ab i32)

        ;; Again, several ways to achieve this, but this function shows one
        ;; way of using local.tee -- in folded form it returns a result as
        ;; well as storing it in a local.
        (i32.add
            (local.get $a)
            (local.tee $ab
                (i32.add (local.get $a) (local.get $b))))

        (local.get $ab)
    )
)