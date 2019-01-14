package vm

import (
	"crypto/sha256"
	"github.com/go-interpreter/wagon/exec"
)

type CosVMExport struct {
	*CosVMNative
}

func (w *CosVMExport) sha256(proc *exec.Process, pSrc int32, lenSrc int32, pDst int32, lenDst int32) {
	srcBuf := w.cosVM.read(proc, pSrc, lenSrc, "sha256().read")
	out := sha256.Sum256(srcBuf)
	w.cosVM.write(proc, out[:], pDst, lenDst, "sha256().write")
}

func (w *CosVMExport) currentBlockNumber(proc *exec.Process) int64 {
	return int64(w.CurrentBlockNumber())
}

func (w *CosVMExport) currentTimestamp(proc *exec.Process) int64 {
	return int64(w.CurrentTimestamp())
}

func (w *CosVMExport) currentWitness(proc *exec.Process, pDst int32, dstSize int32) (length int32) {
	return w.cosVM.write(proc, []byte(w.CurrentWitness()), pDst, dstSize, "currentWitness()")
}

func (w *CosVMExport) printString(proc *exec.Process, pStr int32, lenStr int32) {
	w.PrintString(string(w.cosVM.read(proc, pStr, lenStr, "printString()")))
}

func (w *CosVMExport) printInt64(proc *exec.Process, value int64) {
	w.PrintInt64(value)
}

func (w *CosVMExport) printUint64(proc *exec.Process, value int64) {
	w.PrintUint64(uint64(value))
}

func (w *CosVMExport) requiredAuth(proc *exec.Process, pStr int32, pLen int32) {
	w.RequiredAuth(string(w.cosVM.read(proc, pStr, pLen, "requiredAuth()")))
}

func (w *CosVMExport) getBalanceByName(proc *exec.Process, ptr int32, len int32) int64 {
	return int64(w.GetBalanceByName(string(w.cosVM.read(proc, ptr, len, "getBalanceByName()"))))
}

func (w *CosVMExport) getContractBalance(proc *exec.Process, cPtr int32, cLen int32, nPtr int32, nLen int32) int64 {
	return int64(w.GetContractBalance(
		string(w.cosVM.read(proc, cPtr, cLen, "getContractBalance().contract")),
		string(w.cosVM.read(proc, nPtr, nLen, "getContractBalance().owner")),
	))
}

func (w *CosVMExport) saveToStorage(proc *exec.Process, pKey int32, kLen int32, pValue int32, vLen int32) {
	w.SaveToStorage(
		w.cosVM.read(proc, pKey, kLen, "saveToStorage().key"),
		w.cosVM.read(proc, pValue, vLen, "saveToStorage().value"),
	)
}

func (w *CosVMExport) readFromStorage(proc *exec.Process, pKey int32, kLen int32, pValue int32, vLen int32) int32 {
	return w.cosVM.write(
		proc,
		w.ReadFromStorage(w.cosVM.read(proc, pKey, kLen, "readFromStorage().key")),
		pValue,
		vLen,
		"readFromStorage().value",
	)
}

func (w *CosVMExport) cosAssert(proc *exec.Process, condition int32, pStr int32, len int32) {
	w.CosAssert(condition != 0, string(w.cosVM.read(proc, pStr, len, "cosAssert().msg")))
}

func (w *CosVMExport) cosAbort(proc *exec.Process) {
	w.CosAbort()
}

func (w *CosVMExport) readContractOpParams(proc *exec.Process, ptr, length int32) int32 {
	return w.cosVM.write(proc, []byte(w.ReadContractOpParams()), ptr, length, "readContractOpParams()")
}

func (w *CosVMExport) readContractName(proc *exec.Process, pStr int32, length int32) int32 {
	return w.cosVM.write(proc, []byte(w.ReadContractName()), pStr, length, "readContractName()")
}

func (w *CosVMExport) readContractMethod(proc *exec.Process, pStr int32, length int32) int32 {
	return w.cosVM.write(proc, []byte(w.ReadContractMethod()), pStr, length, "readContractMethod()")
}

func (w *CosVMExport) readContractOwner(proc *exec.Process, pStr int32, length int32) int32 {
	return w.cosVM.write(proc, []byte(w.ReadContractOwner()), pStr, length, "readContractOwner()")
}

func (w *CosVMExport) readContractCaller(proc *exec.Process, pStr int32, length int32) int32 {
	return w.cosVM.write(proc, []byte(w.ReadContractCaller()), pStr, length, "readContractCaller()")
}

func (w *CosVMExport) contractCalledByUser(proc *exec.Process) int32 {
	r := int32(0)
	if w.ContractCalledByUser() {
		r = 1
	}
	return r
}

func (w *CosVMExport) readCallingContractOwner(proc *exec.Process, pStr int32, length int32) int32 {
	return w.cosVM.write(proc, []byte(w.ReadCallingContractOwner()), pStr, length, "readCallingContractOwner()")
}

func (w *CosVMExport) readCallingContractName(proc *exec.Process, pStr int32, length int32) int32 {
	return w.cosVM.write(proc, []byte(w.ReadCallingContractName()), pStr, length, "readCallingContractName()")
}

func (w *CosVMExport) contractTransferToUser(proc *exec.Process, pTo, pToLen int32, amount int64, pMemo, pMemoLen int32) {
	w.ContractTransferToUser(string(w.cosVM.read(proc, pTo, pToLen, "contractTransferToUser().to")), uint64(amount))
}

func (w *CosVMExport) contractTransferToContract(proc *exec.Process, pToOwner, pToOwnerLen, pToContract, pToContractLen int32, amount int64, pMemo, pMemoLen int32) {
	w.ContractTransferToContract(
		string(w.cosVM.read(proc, pToOwner, pToOwnerLen, "contractTransferToContract().toOwner")),
		string(w.cosVM.read(proc, pToContract, pToContractLen, "contractTransferToContract().toContract")),
		uint64(amount))
}

func (w *CosVMExport) readContractSenderValue(proc *exec.Process) int64 {
	return int64(w.ReadContractSenderValue())
}

func (w *CosVMExport) contractCall(proc *exec.Process, owner, ownerSize, contract, contractSize, method, methodSize, param, paramSize int32, coins int64) {
	w.ContractCall(
		string(w.cosVM.read(proc, owner, ownerSize, "contractCall().owner")),
		string(w.cosVM.read(proc, contract, contractSize, "contractCall().contract")),
		string(w.cosVM.read(proc, method, methodSize, "contractCall().method")),
		w.cosVM.read(proc, param, paramSize, "contractCall().param"),
		uint64(coins),
		)
}

func (w *CosVMExport) tableGetRecord(proc *exec.Process, tableName, tableNameLen int32, primary, primaryLen int32, value, valueLen int32) int32 {
	return w.cosVM.write(proc,
		w.TableGetRecord(
			string(w.cosVM.read(proc, tableName, tableNameLen, "tableGetRecord().table_name")),
			w.cosVM.read(proc, primary, primaryLen, "tableGetRecord().primary"),
		),
		value, valueLen, "tableGetRecord()")
}

func (w *CosVMExport) tableNewRecord(proc *exec.Process, tableName, tableNameLen int32, value, valueLen int32) {
	w.TableNewRecord(
		string(w.cosVM.read(proc, tableName, tableNameLen, "tableNewRecord().table_name")),
		w.cosVM.read(proc, value, valueLen, "tableNewRecord().value"),
	)
}

func (w *CosVMExport) tableUpdateRecord(proc *exec.Process, tableName, tableNameLen int32, primary, primaryLen int32, value, valueLen int32) {
	w.TableUpdateRecord(
		string(w.cosVM.read(proc, tableName, tableNameLen, "tableUpdateRecord().table_name")),
		w.cosVM.read(proc, primary, primaryLen, "tableUpdateRecord().primary"),
		w.cosVM.read(proc, value, valueLen, "tableUpdateRecord().value"),
	)
}

func (w *CosVMExport) tableDeleteRecord(proc *exec.Process, tableName, tableNameLen int32, primary, primaryLen int32) {
	w.TableDeleteRecord(
		string(w.cosVM.read(proc, tableName, tableNameLen, "tableDeleteRecord().table_name")),
		w.cosVM.read(proc, primary, primaryLen, "tableDeleteRecord().primary"),
	)
}
