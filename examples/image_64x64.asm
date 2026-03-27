; Requires minimum of 64*64*4 = 16,384 Bytes of memory

li a0, 64  ; x
li a1, 64  ; y
li t0, 255 ; Alpha value
loop_y:
    li a0, 64
    loop_x:
        addi sp, sp, -4    ; reserve 1 word
        sb a0, 0(sp)       ; R
        sb zero, 1(sp)     ; G
        sb a1, 2(sp)       ; B
        sb t0, 3(sp)       ; A
        addi a0, a0, -1
        bne a0, zero, loop_x   ; keep looping while a0 != 0
    addi a1, a1, -1
    bne a1, zero, loop_y       ; keep looping while a1 != 0
end
