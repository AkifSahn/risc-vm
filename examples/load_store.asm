    ; This example demonstrates different read and write instructions

main:
    li t0, 255

    ; Reserve 4 bytes
    ; And fill each byte with 255
    addi sp sp -4 
    sb t0 3(sp)
    sb t0 2(sp)
    sb t0 1(sp)
    sb t0 0(sp)

    ; Read byte, half word and full word into seperate register
    lb a0 0(sp)
    lh a1 0(sp)
    lw a2 0(sp)
    addi sp sp 4
