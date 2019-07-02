package vm

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/go-interpreter/wagon/exec"
)

func e_sha256(proc *exec.Process, pSrc int32, lenSrc int32, pDst int32, lenDst int32) int32 {
	w := proc.GetTag().(*CosVMNative)
	srcBuf := w.cosVM.read(proc, pSrc, lenSrc, "sha256().read")
	out := sha256.Sum256(srcBuf)
	return w.cosVM.write(proc, out[:], pDst, lenDst, "sha256().write")
}

func e_currentBlockNumber(proc *exec.Process) int64 {
	w := proc.GetTag().(*CosVMNative)

	return int64(w.CurrentBlockNumber())
}

func e_currentTimestamp(proc *exec.Process) int64 {
	w := proc.GetTag().(*CosVMNative)

	return int64(w.CurrentTimestamp())
}

func e_currentWitness(proc *exec.Process, pDst int32, dstSize int32) (length int32) {
	w := proc.GetTag().(*CosVMNative)

	return w.cosVM.write(proc, []byte(w.CurrentWitness()), pDst, dstSize, "currentWitness()")
}

func e_getBlockProducers(proc *exec.Process, pDst int32, dstSize int32) (length int32) {
	w := proc.GetTag().(*CosVMNative)

	return w.cosVM.write(proc, []byte(w.GetBlockProducers()), pDst, dstSize, "getBlockProducers()")
}

func e_printString(proc *exec.Process, pStr int32, lenStr int32) {
	w := proc.GetTag().(*CosVMNative)

	w.PrintString(string(w.cosVM.read(proc, pStr, lenStr, "printString()")))
}

func e_printInt64(proc *exec.Process, value int64) {
	w := proc.GetTag().(*CosVMNative)

	w.PrintInt64(value)
}

func e_printUint64(proc *exec.Process, value int64) {
	w := proc.GetTag().(*CosVMNative)

	w.PrintUint64(uint64(value))
}

func e_requiredAuth(proc *exec.Process, pStr int32, pLen int32) {
	w := proc.GetTag().(*CosVMNative)

	w.RequiredAuth(string(w.cosVM.read(proc, pStr, pLen, "requiredAuth()")))
}

func e_getUserBalance(proc *exec.Process, ptr int32, len int32) int64 {
	w := proc.GetTag().(*CosVMNative)

	return int64(w.GetUserBalance(string(w.cosVM.read(proc, ptr, len, "getUserBalance()"))))
}

func e_getContractBalance(proc *exec.Process, cPtr int32, cLen int32, nPtr int32, nLen int32) int64 {
	w := proc.GetTag().(*CosVMNative)

	return int64(w.GetContractBalance(
		string(w.cosVM.read(proc, cPtr, cLen, "getContractBalance().contract")),
		string(w.cosVM.read(proc, nPtr, nLen, "getContractBalance().owner")),
	))
}

func e_cosAssert(proc *exec.Process, condition int32, pStr int32, len int32) {
	w := proc.GetTag().(*CosVMNative)

	w.CosAssert(condition != 0, string(w.cosVM.read(proc, pStr, len, "cosAssert().msg")))
}

func e_cosAbort(proc *exec.Process) {
	w := proc.GetTag().(*CosVMNative)

	w.CosAbort()
}

func e_readContractOpParams(proc *exec.Process, ptr, length int32) int32 {
	w := proc.GetTag().(*CosVMNative)

	return w.cosVM.write(proc, []byte(w.ReadContractOpParams()), ptr, length, "readContractOpParams()")
}

func e_readContractName(proc *exec.Process, pStr int32, length int32) int32 {
	w := proc.GetTag().(*CosVMNative)

	return w.cosVM.write(proc, []byte(w.ReadContractName()), pStr, length, "readContractName()")
}

func e_readContractMethod(proc *exec.Process, pStr int32, length int32) int32 {
	w := proc.GetTag().(*CosVMNative)

	return w.cosVM.write(proc, []byte(w.ReadContractMethod()), pStr, length, "readContractMethod()")
}

func e_readContractOwner(proc *exec.Process, pStr int32, length int32) int32 {
	w := proc.GetTag().(*CosVMNative)

	return w.cosVM.write(proc, []byte(w.ReadContractOwner()), pStr, length, "readContractOwner()")
}

func e_readContractCaller(proc *exec.Process, pStr int32, length int32) int32 {
	w := proc.GetTag().(*CosVMNative)

	return w.cosVM.write(proc, []byte(w.ReadContractCaller()), pStr, length, "readContractCaller()")
}

func e_contractCalledByUser(proc *exec.Process) int32 {
	w := proc.GetTag().(*CosVMNative)

	r := int32(0)
	if w.ContractCalledByUser() {
		r = 1
	}
	return r
}

func e_readCallingContractOwner(proc *exec.Process, pStr int32, length int32) int32 {
	w := proc.GetTag().(*CosVMNative)

	return w.cosVM.write(proc, []byte(w.ReadCallingContractOwner()), pStr, length, "readCallingContractOwner()")
}

func e_readCallingContractName(proc *exec.Process, pStr int32, length int32) int32 {
	w := proc.GetTag().(*CosVMNative)

	return w.cosVM.write(proc, []byte(w.ReadCallingContractName()), pStr, length, "readCallingContractName()")
}

func e_contractTransferToUser(proc *exec.Process, pTo, pToLen int32, amount int64, pMemo, pMemoLen int32) {
	w := proc.GetTag().(*CosVMNative)

	w.ContractTransferToUser(string(w.cosVM.read(proc, pTo, pToLen, "contractTransferToUser().to")), uint64(amount))
}

func e_contractTransferToContract(proc *exec.Process, pToOwner, pToOwnerLen, pToContract, pToContractLen int32, amount int64, pMemo, pMemoLen int32) {
	w := proc.GetTag().(*CosVMNative)

	w.ContractTransferToContract(
		string(w.cosVM.read(proc, pToOwner, pToOwnerLen, "contractTransferToContract().toOwner")),
		string(w.cosVM.read(proc, pToContract, pToContractLen, "contractTransferToContract().toContract")),
		uint64(amount))
}

func e_readContractSenderValue(proc *exec.Process) int64 {
	w := proc.GetTag().(*CosVMNative)

	return int64(w.ReadContractSenderValue())
}

func e_contractCall(proc *exec.Process, owner, ownerSize, contract, contractSize, method, methodSize, param, paramSize int32, coins int64) {
	w := proc.GetTag().(*CosVMNative)

	w.ContractCall(
		string(w.cosVM.read(proc, owner, ownerSize, "contractCall().owner")),
		string(w.cosVM.read(proc, contract, contractSize, "contractCall().contract")),
		string(w.cosVM.read(proc, method, methodSize, "contractCall().method")),
		w.cosVM.read(proc, param, paramSize, "contractCall().param"),
		uint64(coins),
		)
}

func e_tableGetRecord(proc *exec.Process, tableName, tableNameLen int32, primary, primaryLen int32, value, valueLen int32) int32 {
	w := proc.GetTag().(*CosVMNative)

	return w.cosVM.write(proc,
		w.TableGetRecord(
			string(w.cosVM.read(proc, tableName, tableNameLen, "tableGetRecord().table_name")),
			w.cosVM.read(proc, primary, primaryLen, "tableGetRecord().primary"),
		),
		value, valueLen, "tableGetRecord()")
}

func e_tableNewRecord(proc *exec.Process, tableName, tableNameLen int32, value, valueLen int32) {
	w := proc.GetTag().(*CosVMNative)

	w.TableNewRecord(
		string(w.cosVM.read(proc, tableName, tableNameLen, "tableNewRecord().table_name")),
		w.cosVM.read(proc, value, valueLen, "tableNewRecord().value"),
	)
}

func e_tableUpdateRecord(proc *exec.Process, tableName, tableNameLen int32, primary, primaryLen int32, value, valueLen int32) {
	w := proc.GetTag().(*CosVMNative)

	w.TableUpdateRecord(
		string(w.cosVM.read(proc, tableName, tableNameLen, "tableUpdateRecord().table_name")),
		w.cosVM.read(proc, primary, primaryLen, "tableUpdateRecord().primary"),
		w.cosVM.read(proc, value, valueLen, "tableUpdateRecord().value"),
	)
}

func e_tableDeleteRecord(proc *exec.Process, tableName, tableNameLen int32, primary, primaryLen int32) {
	w := proc.GetTag().(*CosVMNative)

	w.TableDeleteRecord(
		string(w.cosVM.read(proc, tableName, tableNameLen, "tableDeleteRecord().table_name")),
		w.cosVM.read(proc, primary, primaryLen, "tableDeleteRecord().primary"),
	)
}

func e_tableGetRecordEx(proc *exec.Process, ownerName, ownerNameLen int32, contractName, contractNameLen int32, tableName, tableNameLen int32, primary, primaryLen int32, value, valueLen int32) int32 {
	w := proc.GetTag().(*CosVMNative)

	return w.cosVM.write(proc,
		w.TableGetRecordEx(
			string(w.cosVM.read(proc, ownerName, ownerNameLen, "tableGetRecordEx().owner_name")),
			string(w.cosVM.read(proc, contractName, contractNameLen, "tableGetRecordEx().contract_name")),
			string(w.cosVM.read(proc, tableName, tableNameLen, "tableGetRecordEx().table_name")),
			w.cosVM.read(proc, primary, primaryLen, "tableGetRecordEx().primary"),
		),
		value, valueLen, "tableGetRecordEx()")
}

func e_setReputationAdmin(proc *exec.Process, name, nameLen int32) {
	w := proc.GetTag().(*CosVMNative)

	w.CosAssert(
		w.ReadContractOwner() == constants.COSSysAccount && w.ReadContractCaller() == constants.COSSysAccount,
		"SetReputationAdmin(): access denied",
		)
	w.SetReputationAdmin(string(w.cosVM.read(proc, name, nameLen, "setReputationAdmin().name")))
}

func e_getReputationAdmin(proc *exec.Process, name, nameLen int32) int32 {
	w := proc.GetTag().(*CosVMNative)

	return w.cosVM.write(proc, []byte(w.GetReputationAdmin()), name, nameLen, "getReputationAdmin()")
}

func e_setReputation(proc *exec.Process, namePtrs, namePtrLen, nameSizes, nameSizeLen, reputations, reputationsLen, memoPtrs, memoPtrLen, memoSizes, memoSizeLen int32) int32 {
	w := proc.GetTag().(*CosVMNative)

	w.CosAssert(
		w.ReadContractOwner() == constants.COSSysAccount && w.ReadContractCaller() == w.GetReputationAdmin(),
		"SetReputation(): access denied",
	)

	namePtr := w.cosVM.read(proc, namePtrs, namePtrLen, "setReputation().namePtrs")
	nameSize := w.cosVM.read(proc, nameSizes, nameSizeLen, "setReputation().nameSizes")
	valInts := w.cosVM.read(proc, reputations, reputationsLen, "setReputation().reputations")
	memoPtr := w.cosVM.read(proc, memoPtrs, memoPtrLen, "setReputation().memoPtrs")
	memoSize := w.cosVM.read(proc, memoSizes, memoSizeLen, "setReputation().memoSizes")

	count := int(namePtrLen / 4)
	w.CosAssert(namePtrLen == nameSizeLen && namePtrLen == reputationsLen && namePtrLen == memoPtrLen && namePtrLen == memoSizeLen, "setReputation(): illegal parameters")

	for i := 0; i < count; i++ {
		offset := i * 4
		name := string(w.cosVM.read(proc,
			int32(binary.LittleEndian.Uint32(namePtr[offset:])),
			int32(binary.LittleEndian.Uint32(nameSize[offset:])),
			fmt.Sprintf("setReputation().names[%d]", i),
		))
		value := binary.LittleEndian.Uint32(valInts[offset:])
		w.CosAssert(value >= constants.MinReputation && value <= constants.MaxReputation,
			fmt.Sprintf("setReputation().reputation[%d]=%d: out of bounds", i, value))
		memo := string(w.cosVM.read(proc,
			int32(binary.LittleEndian.Uint32(memoPtr[offset:])),
			int32(binary.LittleEndian.Uint32(memoSize[offset:])),
			fmt.Sprintf("setReputation().memos[%d]", i),
		))
		w.SetUserReputation(name, value, memo)
	}
	return int32(count)
}

func e_memcpy(proc *exec.Process, dst, src, size int32) int32 {
	w := proc.GetTag().(*CosVMNative)

	return w.cosVM.memcpy(proc, dst, src, size)
}

func e_memset(proc *exec.Process, dst, value, size int32) int32 {
	w := proc.GetTag().(*CosVMNative)

	return w.cosVM.memset(proc, dst, value, size)
}

func e_memmove(proc *exec.Process, dst, src, size int32) int32 {
	w := proc.GetTag().(*CosVMNative)

	return w.cosVM.memmove(proc, dst, src, size)}

func e_memcmp(proc *exec.Process, lhs, rhs, size int32) int32 {
	w := proc.GetTag().(*CosVMNative)

	return w.cosVM.memcmp(proc, lhs, rhs, size)
}

func e_copy(proc *exec.Process, src int32, dst int32, length int32) int32 {
	w := proc.GetTag().(*CosVMNative)
	return w.cosVM.copy(proc, dst, src, length)
}