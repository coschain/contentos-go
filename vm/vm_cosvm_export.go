package vm

import "github.com/go-interpreter/wagon/exec"

type CosVMExport struct {
	*CosVMNative
}

func (w *CosVMExport) currentBlockNumber(proc *exec.Process) int64 {
	return int64(w.CurrentBlockNumber())
}

func (w *CosVMExport) currentTimestamp(proc *exec.Process) int64 {
	return int64(w.CurrentTimestamp())
}

func (w *CosVMExport) currentWitness(proc *exec.Process, pDst int32) (length int32) {
	witness := w.CurrentWitness()
	buf := []byte(witness)
	length, err := w.cosVM.writeStrAt(proc, buf, pDst, int32(maxReadLength))
	w.CosAssert(err == nil, "get current witness error")
	return length
}

func (w *CosVMExport) printString(proc *exec.Process, pStr int32, lenStr int32) {
	var str []byte
	_, err := w.cosVM.readStrAt(proc, pStr, lenStr, &str)
	w.CosAssert(err == nil, "read string error when try to print")
	w.PrintString(string(str))
}

func (w *CosVMExport) printUint32(proc *exec.Process, value int32) {
	w.PrintUint32(uint32(value))
}

func (w *CosVMExport) printUint64(proc *exec.Process, value int64) {
	w.PrintUint64(uint64(value))
}

func (w *CosVMExport) printBool(proc *exec.Process, value int32) {
	w.PrintBool(value > 0)
}

func (w *CosVMExport) requiredAuth(proc *exec.Process, pStr int32, pLen int32) {
	var name []byte
	_, err := w.cosVM.readStrAt(proc, pStr, pLen, &name)
	if err != nil {
		panic("read auth name error")
	}
	w.RequiredAuth(string(name))
}

func (w *CosVMExport) getBalanceByName(proc *exec.Process, ptr int32, len int32) int64 {
	var name []byte
	_, err := w.cosVM.readStrAt(proc, ptr, len, &name)
	if err != nil {
		panic(err)
	}
	return int64(w.GetBalanceByName(string(name)))
}

func (w *CosVMExport) getContractBalance(proc *exec.Process, cPtr int32, cLen int32, nPtr int32, nLen int32) int64 {
	var contract []byte
	_, err := w.cosVM.readStrAt(proc, cPtr, cLen, &contract)
	w.CosAssert(err == nil, "get contract balance error when read contract name")
	var name []byte
	_, err = w.cosVM.readStrAt(proc, nPtr, nLen, &name)
	w.CosAssert(err == nil, "get contract balance error when read name")
	return int64(w.GetContractBalance(string(contract), string(name)))
}

func (w *CosVMExport) saveToStorage(proc *exec.Process, pKey int32, kLen int32, pValue int32, vLen int32) {
	var key []byte
	_, err := w.cosVM.readStrAt(proc, pKey, kLen, &key)
	w.CosAssert(err == nil, "read key failed when save to storage")
	var value []byte
	_, err = w.cosVM.readStrAt(proc, pValue, vLen, &value)
	w.CosAssert(err == nil, "read value failed when save to storage")
	w.SaveToStorage(key, value)
}

func (w *CosVMExport) readFromStorage(proc *exec.Process, pKey int32, kLen int32, pValue int32, vLen int32) {
	var key []byte
	_, err := w.cosVM.readStrAt(proc, pKey, kLen, &key)
	w.CosAssert(err == nil, "read key failed when read from storage")
	value := w.ReadFromStorage(key)
	if len(value) > int(vLen) {
		value = value[:vLen]
	}
	_, err = w.cosVM.writeStrAt(proc, value, pValue, vLen)
	w.CosAssert(err == nil, "write value failed when read from storage")
}

func (w *CosVMExport) cosAssert(proc *exec.Process, condition int32, pStr int32, len int32) {
	var msg []byte
	_, err := w.cosVM.readStrAt(proc, pStr, len, &msg)
	if err != nil {
		panic("read msg when assert failed")
	}
	w.CosAssert(condition > 0, string(msg))
}

func (w *CosVMExport) readContractOpParams(proc *exec.Process, ptr, length int32) {
	params := w.ReadContractOpParams()
	b := []byte(params)
	if len(b) > int(length) {
		b = b[:length]
	}
	_, err := w.cosVM.writeStrAt(proc, b, ptr, length)
	w.CosAssert(err == nil, "read contract params error")
}

func (w *CosVMExport) readContractOpParamsLength(proc *exec.Process) int32 {
	length := int32(w.ReadContractOpParamsLength())
	return length
}

func (w *CosVMExport) readContractOwner(proc *exec.Process, pStr int32, length int32) {
	owner := w.ReadContractOwner()
	byteOwner := []byte(owner)
	if len(byteOwner) > int(length) {
		byteOwner = byteOwner[:length]
	}
	_, err := w.cosVM.writeStrAt(proc, byteOwner, pStr, length)
	w.CosAssert(err == nil, "write owner into memory err")
}

func (w *CosVMExport) readContractCaller(proc *exec.Process, pStr int32, length int32) {
	caller := w.ReadContractCaller()
	byteCaller := []byte(caller)
	if len(byteCaller) > int(length) {
		byteCaller = byteCaller[:length]
	}
	_, err := w.cosVM.writeStrAt(proc, byteCaller, pStr, length)
	w.CosAssert(err == nil, "read contract caller error")
}

func (w *CosVMExport) contractTransfer(proc *exec.Process, pTo, pToLen int32, amount int64) {
	var to []byte
	_, err := w.cosVM.readStrAt(proc, pTo, pToLen, &to)
	w.CosAssert(err == nil, "read to err when transfer")
	w.ContractTransfer(string(to), uint64(amount))
}

func (w *CosVMExport) getSenderValue(proc *exec.Process) int64 {
	return int64(w.GetSenderValue())
}
