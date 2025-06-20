package main

import (
	"runtime"

	"github.com/Testzyler/order-management-go/cmd"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	cmd.Execute()
}
