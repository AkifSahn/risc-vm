main:
    li      a5,10
    .L2:
    addi    a5,a5,-1

    addi sp sp -4
    sw   a5 0(sp) ; store the counter

    bgt a5,zero,.L2
    end
