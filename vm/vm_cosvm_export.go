package vm

import (
	"crypto/sha256"
	"fmt"
	"github.com/go-interpreter/wagon/exec"
)

type CosVMExport struct {
	*CosVMNative
}

func (w *CosVMExport) read(proc *exec.Process, buffer int32, bufferSize int32, tag string) []byte {
	w.CosAssert(bufferSize >= 0, fmt.Sprintf("%s: negative reading size", tag))
	buf := make([]byte, bufferSize)
	n, err := proc.ReadAt(buf, int64(buffer))
	w.CosAssert(err == nil, fmt.Sprintf("%s: reading failed. %v", tag, err))
	return buf[:n]
}

func (w *CosVMExport) write(proc *exec.Process, data []byte, buffer int32, bufferSize int32, tag string) int32 {
	size := int32(len(data))
	if bufferSize <= 0 {
		return size
	}
	if size > bufferSize {
		size = bufferSize
	}
	n, err := proc.WriteAt(data[:size], int64(buffer))
	w.CosAssert(err == nil, fmt.Sprintf("%s: writing failed. %v", tag, err))
	return int32(n)
}

func (w *CosVMExport) sha256(proc *exec.Process, pSrc int32, lenSrc int32, pDst int32, lenDst int32) {
	srcBuf := w.read(proc, pSrc, lenSrc, "sha256().read")
	out := sha256.Sum256(srcBuf)
	w.write(proc, out[:], pDst, lenDst, "sha256().write")
}

func (w *CosVMExport) currentBlockNumber(proc *exec.Process) int64 {
	return int64(w.CurrentBlockNumber())
}

func (w *CosVMExport) currentTimestamp(proc *exec.Process) int64 {
	return int64(w.CurrentTimestamp())
}

func (w *CosVMExport) currentWitness(proc *exec.Process, pDst int32, dstSize int32) (length int32) {
	return w.write(proc, []byte(w.CurrentWitness()), pDst, dstSize, "currentWitness()")
}

func (w *CosVMExport) printString(proc *exec.Process, pStr int32, lenStr int32) {
	w.PrintString(string(w.read(proc, pStr, lenStr, "printString()")))
}

func (w *CosVMExport) printInt64(proc *exec.Process, value int64) {
	w.PrintInt64(value)
}

func (w *CosVMExport) printUint64(proc *exec.Process, value int64) {
	w.PrintUint64(uint64(value))
}

func (w *CosVMExport) requiredAuth(proc *exec.Process, pStr int32, pLen int32) {
	w.RequiredAuth(string(w.read(proc, pStr, pLen, "requiredAuth()")))
}

func (w *CosVMExport) getBalanceByName(proc *exec.Process, ptr int32, len int32) int64 {
	return int64(w.GetBalanceByName(string(w.read(proc, ptr, len, "getBalanceByName()"))))
}

func (w *CosVMExport) getContractBalance(proc *exec.Process, nPtr int32, nLen int32, cPtr int32, cLen int32) int64 {
	return int64(w.GetContractBalance(
		string(w.read(proc, cPtr, cLen, "getContractBalance().contract")),
		string(w.read(proc, nPtr, nLen, "getContractBalance().owner")),
	))
}

func (w *CosVMExport) saveToStorage(proc *exec.Process, pKey int32, kLen int32, pValue int32, vLen int32) {
	w.SaveToStorage(
		w.read(proc, pKey, kLen, "saveToStorage().key"),
		w.read(proc, pValue, vLen, "saveToStorage().value"),
	)
}

func (w *CosVMExport) readFromStorage(proc *exec.Process, pKey int32, kLen int32, pValue int32, vLen int32) int32 {
	return w.write(
		proc,
		w.ReadFromStorage(w.read(proc, pKey, kLen, "readFromStorage().key")),
		pValue,
		vLen,
		"readFromStorage().value",
	)
}

func (w *CosVMExport) cosAssert(proc *exec.Process, condition int32, pStr int32, len int32) {
	w.CosAssert(condition != 0, string(w.read(proc, pStr, len, "cosAssert().msg")))
}

func (w *CosVMExport) cosAbort(proc *exec.Process) {
	w.CosAbort()
}

func (w *CosVMExport) readContractOpParams(proc *exec.Process, ptr, length int32) int32 {
	return w.write(proc, []byte(w.ReadContractOpParams()), ptr, length, "readContractOpParams()")
}

func (w *CosVMExport) readContractName(proc *exec.Process, pStr int32, length int32) int32 {
	return w.write(proc, []byte(w.ReadContractName()), pStr, length, "readContractName()")
}

func (w *CosVMExport) readContractMethod(proc *exec.Process, pStr int32, length int32) int32 {
	return w.write(proc, []byte(w.ReadContractMethod()), pStr, length, "readContractMethod()")
}

func (w *CosVMExport) readContractOwner(proc *exec.Process, pStr int32, length int32) int32 {
	return w.write(proc, []byte(w.ReadContractOwner()), pStr, length, "readContractOwner()")
}

func (w *CosVMExport) readContractCaller(proc *exec.Process, pStr int32, length int32) int32 {
	return w.write(proc, []byte(w.ReadContractCaller()), pStr, length, "readContractCaller()")
}

func (w *CosVMExport) contractTransfer(proc *exec.Process, pTo, pToLen int32, amount int64, pMemo, pMemoLen int32) {
	w.ContractTransfer(string(w.read(proc, pTo, pToLen, "contractTransfer().to")), uint64(amount))
}

func (w *CosVMExport) readContractSenderValue(proc *exec.Process) int64 {
	return int64(w.ReadContractSenderValue())
}

func (w *CosVMExport) tableGetRecord(proc *exec.Process, tableName, tableNameLen int32, primary, primaryLen int32, value, valueLen int32) int32 {
	return w.write(proc,
		w.TableGetRecord(
			string(w.read(proc, tableName, tableNameLen, "tableGetRecord().table_name")),
			w.read(proc, primary, primaryLen, "tableGetRecord().primary"),
		),
		value, valueLen, "tableGetRecord()")
}

func (w *CosVMExport) tableNewRecord(proc *exec.Process, tableName, tableNameLen int32, value, valueLen int32) {
	w.TableNewRecord(
		string(w.read(proc, tableName, tableNameLen, "tableNewRecord().table_name")),
		w.read(proc, value, valueLen, "tableNewRecord().value"),
	)
}

func (w *CosVMExport) tableUpdateRecord(proc *exec.Process, tableName, tableNameLen int32, primary, primaryLen int32, value, valueLen int32) {
	w.TableUpdateRecord(
		string(w.read(proc, tableName, tableNameLen, "tableUpdateRecord().table_name")),
		w.read(proc, primary, primaryLen, "tableUpdateRecord().primary"),
		w.read(proc, value, valueLen, "tableUpdateRecord().value"),
	)
}

func (w *CosVMExport) tableDeleteRecord(proc *exec.Process, tableName, tableNameLen int32, primary, primaryLen int32) {
	w.TableDeleteRecord(
		string(w.read(proc, tableName, tableNameLen, "tableDeleteRecord().table_name")),
		w.read(proc, primary, primaryLen, "tableDeleteRecord().primary"),
	)
}
