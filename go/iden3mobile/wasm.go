package iden3mobile

import (
	"fmt"
	"io/ioutil"

	// "github.com/perlin-network/life/exec"

	// wasm "github.com/wasmerio/go-ext-wasm/wasmer"
	"github.com/matiasinsaurralde/go-wasm3"
)

// // LIFE
// func (i *Identity) CallWasm(filePath string) error {
// 	dat, err := ioutil.ReadFile(filePath)
// 	if err != nil {
// 		return err
// 	}
// 	vm, err := exec.NewVirtualMachine(dat, exec.VMConfig{}, &exec.NopResolver{}, nil)
// 	if err != nil { // if the wasm bytecode is invalid
// 		return err
// 	}
// 	entryID, ok := vm.GetFunctionExport("sum") // can be changed to your own exported function
// 	if !ok {
// 		return errors.New("entry function not found")
// 	}
// 	ret, err := vm.Run(entryID, 3, 4)
// 	if err != nil {
// 		vm.PrintStackTrace()
// 		return err
// 	}
// 	fmt.Printf("return value = %d\n", ret)
// 	return nil
// }

// // WASMER
// func (i *Identity) CallWasm(filePath string) error {
// 	// Reads the WebAssembly module as bytes.
// 	bytes, err := wasm.ReadBytes(filePath)
// 	if err != nil {
// 		return err
// 	}

// 	// Instantiates the WebAssembly module.
// 	instance, err := wasm.NewInstance(bytes)
// 	if err != nil {
// 		return err
// 	}
// 	defer instance.Close()

// 	// Gets the `sum` exported function from the WebAssembly instance.
// 	sum := instance.Exports["sum"]

// 	// Calls that exported function with Go standard values. The WebAssembly
// 	// types are inferred and values are casted automatically.
// 	result, err := sum(5, 37)
// 	if err != nil {
// 		return err
// 	}

// 	fmt.Println(result)
// 	return nil
// }

// WASM3
func (i *Identity) CallWasm(filePath string) error {
	runtime := wasm3.NewRuntime(&wasm3.Config{
		Environment: wasm3.NewEnvironment(),
		StackSize:   64 * 1024,
	})

	wasmBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	module, err := runtime.ParseModule(wasmBytes)
	if err != nil {
		return err
	}
	_, err = runtime.LoadModule(module)
	if err != nil {
		return err
	}

	fn, err := runtime.FindFunction("sum")
	if err != nil {
		return err
	}
	result, err := fn(1, 1)
	fmt.Println(result)
	return err
}
