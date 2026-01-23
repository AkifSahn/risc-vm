addi R1 R0 0
addi R2 R0 80
subi R2 R2 1

addi R3 R0 0
addi R4 R0 1

add R5 R3 R4
add R3 R0 R4
add R4 R0 R5
addi R1 R1 1
blt R1 R2 -5

store 0 R0 R4
end
