factorial:
        li      a5,1
        ble     a0,a5,.L4
        add     a4,a0,a5
        mv      a0,a5
.L3:
        mul     a0,a0,a5
        addi    a5,a5,1
        bne     a5,a4,.L3
        ret
.L4:
        li      a0,1
        ret

main:
        addi    sp,sp,-16
        sw      ra,12(sp)
        li      a0,5     ; input parameter
        call    factorial ; result stored at a0
        lw      ra,12(sp)
        addi    sp,sp,16
        sw      a0 -4(sp) ; store result at memorh
        end
