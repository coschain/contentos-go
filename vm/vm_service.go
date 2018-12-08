package vm

type WasmVmService struct {
}

func (w *WasmVmService) Run( ctx *Context ) error{
	return ctx.Run()
}
