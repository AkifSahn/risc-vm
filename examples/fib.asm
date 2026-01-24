addi s4 zero 0 
addi s5 zero 1 

addi s2 zero 0 
store 0 s2 s4

addi s2 zero 1 
store 0 s2 s5

addi s2 zero 2 
addi s3 zero 11

add s6 s4 s5
store 0 s2 s6
add s4 zero s5
add s5 zero s6
addi s2 s2 1 

blt s2 s3 -6

end

