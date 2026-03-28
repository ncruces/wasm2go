;; Demonstrates WAT code patterns for writing loops of different kinds that
;; would be familiar to C (and other C-based language) programmers.
;;
;; Eli Bendersky [https://eli.thegreenplace.net]
;; This code is in the public domain.
(module
    ;; Memory buffer with data imported from the host.
    (import "env" "buffer" (memory 80))

    (import "env" "log_i32" (func $log_i32 (param i32)))
    (import "env" "rand_i32" (func $rand_i32 (result i32)))

    ;; Adds together elements in memory, starting at offset $start (byte offset
    ;; into linear memory), adds the first $count elements, returning the
    ;; result. Assumes each element is an i32 occupying 4 bytes.
    (func (export "add_all") (param $start i32) (param $count i32) (result i32)
        (local $i i32)
        (local $read_offset i32)
        (local $result i32)

        ;; Equivalent to the C loop:
        ;;
        ;; for (i = 0; i < count; i++) {
        ;;   result += memory[i * 4]
        ;; }
        ;;
        ;; We use a block to enable breaking out of the loop early, and not
        ;; just at the end. This is because a `for` loop performs its test
        ;; before each iteration, not running at all if the condition isn't
        ;; satisfied for the very first i.
        ;;
        ;; The loop is transformed to:
        ;;
        ;; for (;;) {
        ;;   if (i >= count) {
        ;;     break
        ;;   }
        ;;   ... loop body
        ;;   i++
        ;; }
        ;;
        ;; Since the branch to $breakaddloop skips the loop's "br $addloop"
        ;; back-edge, it exits the loop.
        (local.set $i (i32.const 0))
        (loop $addloop (block $breakaddloop
            (i32.ge_s (local.get $i) (local.get $count))
            br_if $breakaddloop

            ;; Calculate read offset of element i in linear memory.
            (local.set $read_offset
                (i32.add
                    (local.get $start)
                    (i32.mul (local.get $i) (i32.const 4))))

            (local.set $result
                (i32.add
                    (local.get $result)
                    (i32.load (local.get $read_offset))))

            (local.set $i (i32.add (local.get $i) (i32.const 1)))
            br $addloop
        ))

        (local.get $result)
    )

    ;; Returns a random number that's a multiple of 10, using rand_i32 which
    ;; just returns random integers.
    ;; Note: this isn't the most mathematically or algorithmically efficient
    ;; way to do this, it's just to illustrate a code pattern.
    (func (export "rand_multiple_of_10") (result i32)
        (local $n i32)

        ;; Equivalent to the C loop:
        ;;
        ;; do {
        ;;   n = rand_i32();
        ;; } while (n % 10 != 0)
        ;;
        ;; The branch is at the end, not looping back unless a condition is
        ;; satisfied.
        (loop $randloop
            (local.set $n (call $rand_i32))

            (i32.ne (i32.rem_u (local.get $n) (i32.const 10)) (i32.const 0))
            br_if $randloop
        )

        (local.get $n)
    )

    ;; Calculates the first power of base that's > limit; e.g. for base=2 and
    ;; limit=1000, the first power that's over the limit is 1024.
    (func (export "first_power_over_limit") (param $base i32) (param $limit i32) (result i32)
        (local $n i32)
        (local.set $n (i32.const 1))

        ;; Equivalent to the C loop:
        ;;
        ;; while (n <= limit) {
        ;;   n *= base
        ;; }
        ;;
        ;; Like in the 'for' loop case, we don't want to run the loop's body
        ;; if the condition is not satisfied from the start; this is why the
        ;; breaking branch is at the very start. The similarity to the 'for'
        ;; loop is not surprising because a 'for' loop is just syntactic sugar
        ;; for a 'while' loop (or the other way around!)
        (loop $powerloop (block $breakpowerloop
            (i32.gt_s (local.get $n) (local.get $limit))
            br_if $breakpowerloop

            (local.set $n (i32.mul (local.get $n) (local.get $base)))
            br $powerloop
        ))

        (local.get $n)
    )
)