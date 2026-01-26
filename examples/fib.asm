main:
    li      a0,30
    jal     ra,fib
    end

fib:
    mv      a4,a0
    bge     zero,a0,.L4
    li      a5,1
    beq     a0,a5,.L1
    li      a5,2
    li      a0,1
    li      a3,0
    .L3:
    mv      a2,a0
    add     a0,a0,a3
    addi    a5,a5,1
    mv      a3,a2
    bge     a4,a5,.L3
    ret
    .L4:
    li      a0,0
    .L1:
    ret
