; C code used to generate this assembly code.
; compiled with '-O1' optimization flag
;
; void swap(int* arr, int a, int b){
;     int temp = arr[a];
;     arr[a] = arr[b];
;     arr[b] = temp;
; }
; 
; void sort(int* arr, int count){
;     for (int i = 0; i < count; ++i){
;         int min_index = i;
;         for (int j = i+1; j < count; ++j){
;             if (arr[j] < arr[min_index]) min_index = j;
;         }
;         swap(arr, i, min_index);
;     }
; }
; 
; int main(){
;     int count = 10;
;     int a[count];
;     for (int i = 0; i < count; ++i){
;         a[i] = count-i;
;     }
;     sort(a, count);
; }

swap:
        slli    a1,a1,2
        add     a1,a0,a1
        lw      a5,0(a1)
        slli    a2,a2,2
        add     a0,a0,a2
        lw      a4,0(a0)
        sw      a4,0(a1)
        sw      a5,0(a0)
        ret
sort:
        ble     a1,zero,.L11
        addi    sp,sp,-32
        sw      ra,28(sp)
        sw      s0,24(sp)
        sw      s1,20(sp)
        sw      s2,16(sp)
        sw      s3,12(sp)
        mv      s1,a0
        mv      s0,a1
        mv      s3,a0
        li      a1,0
        j       .L7
.L5:
        addi    a4,a4,1
        addi    a3,a3,4
        beq     s0,a4,.L14
.L6:
        slli    a5,a2,2
        add     a5,s1,a5
        lw      a6,4(a3)
        lw      a5,0(a5)
        bge     a6,a5,.L5
        mv      a2,a4
        j       .L5
.L14:
        mv      a0,s1
        call    swap
        addi    s3,s3,4
        mv      a1,s2
.L7:
        addi    s2,a1,1
        beq     s0,s2,.L4
        mv      a3,s3
        mv      a4,s2
        mv      a2,a1
        j       .L6
.L4:
        mv      a2,a1
        mv      a0,s1
        call    swap
        lw      ra,28(sp)
        lw      s0,24(sp)
        lw      s1,20(sp)
        lw      s2,16(sp)
        lw      s3,12(sp)
        addi    sp,sp,32
        jr      ra
.L11:
        ret
main:
        addi    sp,sp,-64
        sw      ra,60(sp)
        li      a5,10
        sw      a5,8(sp)
        li      a5,9
        sw      a5,12(sp)
        li      a5,8
        sw      a5,16(sp)
        li      a5,7
        sw      a5,20(sp)
        li      a5,6
        sw      a5,24(sp)
        li      a5,5
        sw      a5,28(sp)
        li      a5,4
        sw      a5,32(sp)
        li      a5,3
        sw      a5,36(sp)
        li      a5,2
        sw      a5,40(sp)
        li      a5,1
        sw      a5,44(sp)
        li      a1,10
        addi    a0,sp,8
        call    sort
        li      a0,0
        lw      ra,60(sp)
        addi    sp,sp,64
        jr      ra
