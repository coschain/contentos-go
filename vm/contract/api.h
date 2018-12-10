

typedef long uint64;
typedef char* ptr;
typedef int len;
typedef int uint32;

extern "C" {

	uint64 current_block_number();

	uint64 current_timestamp();

	void current_witness(ptr , len);

	void sha256(ptr, len, ptr, len);

	void print_str(ptr);

	void print_uint32(uint32);
	void print_uint64(uint64);
	void print_bool(uint32);

	void require_auth(ptr);

	uint64 get_balance_by_name(ptr);

	uint64 get_contract_balance(ptr , ptr);

	void save_to_storage(ptr, len, ptr, len);

	void read_from_storage(ptr, len, ptr, len);

	void log_sort(uint32, ptr, len, ptr, len);

	void cos_assert(bool, ptr );

	void read_contract_op_params( ptr, len, ptr, len);
	len	read_contract_op_params_length();

	void read_contract_owner(ptr, len);
	void read_contract_caller(ptr, len);

	void transfer( ptr , ptr , uint64, ptr );
	uint64 get_sender_value();
}
