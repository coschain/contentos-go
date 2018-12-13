package vm

type HostFunction interface {
	CurrentBlockNumber() uint64
	CurrentTimestamp() uint64
	CurrentWitness() string
	Sha256(in []byte) []byte
	PrintString(str string)
	PrintUint32(v uint32)
	PrintUint64(v uint32)
	PrintBool(v bool)
	RequiredAuth(name string)
	GetBalanceByName(name string) uint64
	GetContractBalance(contract string, name string) uint64
	SaveToStorage(key []byte, value []byte)
	ReadFromStorage(key []byte) []byte
	LogSort(namespace uint32, key []byte, value []byte)
	CosAssert(v bool, info string)
	ReadContractOpParams() []byte
	ReadContractOpParamsLength() uint32
	ReadContractOwner() string
	ReadContractCaller() string
	Transfer(from string, to string, amount uint64, memo string)
	GetSenderValue() uint64
}

//type ptr uint32
//type len uint32
//
//type FunctionRouter interface {
//	current_block_number() uint64
//	current_timestamp() uint64
//	current_witness(ptr, len)
//	sha256(ptr, len, ptr, len)
//	print_str(ptr, len)
//	print_uint32(uint32)
//	print_uint64(uint64)
//	print_bool(uint32)
//	require_auth(ptr)
//	get_balance_by_name(ptr) uint64
//	get_contract_balance(ptr, ptr) uint64
//	save_to_storage(ptr, len, ptr, len)
//	read_from_storage(ptr, len, ptr, len)
//	log_sort(uint32, ptr, len, ptr, len)
//	cos_assert(bool, ptr)
//
//	read_contract_op_params(ptr, len, ptr, len)
//	read_contract_op_params_length() len
//
//	read_contract_owner(ptr, len)
//	read_contract_caller(ptr, len)
//
//	transfer(ptr, ptr, uint64, ptr)
//	get_sender_value() uint64
//}
