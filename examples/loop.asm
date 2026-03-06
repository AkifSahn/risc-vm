main:
    li      a5,0
    li      t0,10
    .L2:
    addi    a5,a5,1
    addi sp sp -4
    sw   a5 0(sp) ; store the counter
    blt a5,t0,.L2
    end
