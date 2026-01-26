    li a0 20

    mv t0 a0
    addi t0 t0 -2
    li t1 1
    li t2 2

    addi t0 t0 -1
    add t1 t1 t2

    xor t1 t1 t2
    xor t2 t1 t2
    xor t1 t1 t2
    bne t0 zero -5

    mv a0 t2
    sw t2 10(zero)
    end
