main:
    li      a0,5
    jal     ra factorial
    sw      a0 0(zero)
    end

factorial:
    blt     a0,zero,.L4
    li      a5,1
    ble     a0,a5,.L5
    add     a4,a0,a5
    li      a5,2
    li      a0,1
    .L3:
    mul     a0,a0,a5
    addi    a5,a5,1
    bne     a5,a4,.L3
    ret
    .L4:
    li      a0,-1
    ret
    .L5:
    li      a0,1
    ret
