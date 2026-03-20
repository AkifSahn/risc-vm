; Requires minimum of 64*64*4*4 = 65,536 Bytes of memory

li a0, 64       ; x
li a1, 64       ; y
li t0, 255 ; temp reg to store maximum pixel value
loop_y:
    li a0, 64
    loop_x:
        addi sp, sp, -16    ; reserve 4 words
        sw a0, 0(sp)        ; R
        sw zero, 4(sp)        ; G
        sw a1, 8(sp)        ; B
        sw t0, 12(sp)       ; A
        addi a0, a0, -1
        bne a0, zero, loop_x   ; keep looping while a0 != 0
    addi a1, a1, -1
    bne a1, zero, loop_y       ; keep looping while a1 != 0
end
