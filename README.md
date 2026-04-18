# RISC-V Vm

### Installing and building

- Clone the repository.
- Ensure you have `go 1.22.3` installed in your machine.
- You can build the project with `go build`.

### Running a RISC-V assembly program

- You can choose an example from `examples/` or just write your own.
- You can use [www.godbolt.org](https://godbolt.org/) to compile **C** code to **Risc-V** assembly code. Just be sure to choose **32-bit** version for compiler.
- To run the program, simply run the simulator with `-file <path_of_the_file>` flag. The simulator will parse and simulate the program and will dump the register, stack and runtime statistics.

> **_NOTE:_**  Currently, Risc-V instruction-set is **not fully** implemented. There may be **unrecognised instructions** or **bugs**.

