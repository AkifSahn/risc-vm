main:
    addi    sp,sp,-128
    sw      ra,124(sp)
    li      a5,1
    sw      a5,76(sp)
    li      a5,2
    sw      a5,80(sp)
    li      a5,3
    sw      a5,84(sp)
    li      a5,4
    sw      a5,88(sp)
    li      a5,5
    sw      a5,92(sp)
    li      a5,6
    sw      a5,96(sp)
    li      a5,7
    sw      a5,100(sp)
    li      a5,8
    sw      a5,104(sp)
    li      a5,9
    sw      a5,108(sp)
    li      a5,10
    sw      a5,40(sp)
    li      a5,11
    sw      a5,44(sp)
    li      a5,12
    sw      a5,48(sp)
    li      a5,13
    sw      a5,52(sp)
    li      a5,14
    sw      a5,56(sp)
    li      a5,15
    sw      a5,60(sp)
    li      a5,16
    sw      a5,64(sp)
    li      a5,17
    sw      a5,68(sp)
    li      a5,18
    sw      a5,72(sp)
    addi    a5,sp,4
    addi    a4,sp,40
    addi    a3,sp,76
    li      a2,3
    mv      a1,a2
    mv      a0,a2
    jal ra    matmul
    lw      a0,4(sp)
    lw      ra,124(sp)
    addi    sp,sp,128
    end
matmul:
    ble     a0,zero,.L13
    addi    sp,sp,-16
    sw      s0,12(sp)
    sw      s1,8(sp)
    sw      s2,4(sp)
    mv      t2,a0
    mv      a6,a1
    mv      t6,a2
    mv      t5,a3
    mv      s1,a4
    mv      t0,a5
    slli    a7,a2,2
    slli    s2,a1,2
    li      s0,0
    j       .L3
    .L6:
    mv      a0,t1
    sw      zero,0(t1)
    ble     a6,zero,.L4
    mv      a1,t4
    mv      a2,t5
    li      a3,0
    .L5:
    lw      a4,0(a2)
    lw      a5,0(a1)
    mul     a4,a4,a5
    lw      a5,0(a0)
    add     a5,a5,a4
    sw      a5,0(a0)
    addi    a3,a3,1
    addi    a2,a2,4
    add     a1,a1,a7
    bne     a6,a3,.L5
    .L4:
    addi    t3,t3,1
    addi    t1,t1,4
    addi    t4,t4,4
    bne     t6,t3,.L6
    .L8:
    addi    s0,s0,1
    add     t0,t0,a7
    add     t5,t5,s2
    beq     t2,s0,.L1
    .L3:
    mv      t4,s1
    mv      t1,t0
    li      t3,0
    bgt     t6,zero,.L6
    j       .L8
    .L1:
    lw      s0,12(sp)
    lw      s1,8(sp)
    lw      s2,4(sp)
    addi    sp,sp,16
    jr      ra
    .L13:
    ret
