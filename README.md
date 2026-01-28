# RISC-V Vm

### Installing and running

- Clone the repository.
- Ensure you have `go 1.22.3` installed in your machine.
- You can build the project with `go build` and then run the generated executable.
- Or just run `go run .` to run the VM with the currently **loaded program**.


### Changing the loaded program

- Loaded program can be changed by changing the `<file_name>` argument given to the `machine.LoadProgramFromFile(<file_name>)` method in the `main.go`'s `main` function.
- You can choose an example from `examples/` or just write your own.
- You can use [www.godbolt.org](https://godbolt.org/) to compile **C** code to **Risc-V** assembly code. Just be sure to choose **32-bit** version for compiler.

> **_NOTE:_**  Currently, Risc-V instruction-set is **not fully** implemented. There may be **unrecognised instructions** or **bugs**.
