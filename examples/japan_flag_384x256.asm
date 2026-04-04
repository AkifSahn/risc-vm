; Requires minimum of 384*256*4 = 393,216 Bytes of memory

main:
    ; Background color
    li s0, 0 
    ori s0, s0, 255 ; A
    slli s0 s0 8
    ori s0, s0, 255 ; B
    slli s0 s0 8
    ori s0, s0, 255 ; G
    slli s0 s0 8
    ori s0, s0, 255 ; R

    ; Circle Color
    ori s1, s1, 255 ; A
    slli s1 s1 8
    ori s1, s1, 45 ; B
    slli s1 s1 8
    ori s1, s1,  0; G
    slli s1 s1 8
    ori s1, s1, 255 ; R

    li s2, 384 ; Width
    li s3, 256 ; Height

    srli a0, s2, 1 ; center-x
    srli a1, s3, 1 ; center-y

    li t0, 5
    div a2, s2, t0 ; radius
    mul a2, a2, a2 ; squared radius

    li a3, 0 ; y-value
    loop_y:
        li a4, 0 ; x-value
        ; calculate y-distance square
        sub t5, a3, a1
        mul t5, t5, t5
        loop_x:
            addi sp, sp, -4    ; reserve 1 word(pixel)

            ; calculate x-distance square
            sub t6, a4, a0
            mul t6, t6, t6

            ; squared distance to the center
            add t0, t5, t6

            ; pixel is outside of the circle
            bgt t0, a2, background

            circle:
                sw s1, 0(sp)
                j .L1
            background:
                sw s0, 0(sp)
            .L1:
                addi a4, a4, 1
                blt a4, s2, loop_x   ; keep looping while a4 < width

        addi a3, a3, 1
        blt a3, s3, loop_y       ; keep looping while a3 < height
