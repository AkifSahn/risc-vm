main:
    addi    sp,sp,-16
    sw      ra,12(sp)
    li      a0,5
    jal     ra factorial
    lw      ra,12(sp)
    addi    sp,sp,16
    end

factorial:
    li      a5,1
    ble     a0,a5,.L3
    addi    sp,sp,-16
    sw      ra,12(sp)
    sw      s0,8(sp)
    mv      s0,a0
    addi    a0,a0,-1
    jal     ra factorial
    mul     a0,a0,s0
    lw      ra,12(sp)
    lw      s0,8(sp)
    addi    sp,sp,16
    jr      ra
    .L3:
    li      a0,1
    ret

