#pragma once

#ifdef __cplusplus
extern "C" {
#endif

    /**
     Get current block number.
     @return The block number.
     */
    unsigned long long current_block_number();

    /**
     Get current timestamp.
     @return The UTC timestamp of head block, in seconds.
     */
    unsigned long long current_timestamp();

    /**
     Get current witness account name.
     @param[in,out] buffer the buffer to which the account name string is stored.
     @param[in] size the capacity of @p buffer, in bytes.
     @return if @p size is positive, return the number of bytes written to @p buffer.
     if @p size is zero or negative, return the length of the account name in bytes without changing @p buffer.
     */
    int current_witness(char* buffer, int size);

    /**
     Get SHA256 hash digest of a blob.
     @param[in] buffer the data to be hashed.
     @param[in] size size of data in bytes.
     @param[in,out] hash the buffer to which digest bytes are stored.
     @param[in] hash_size the capacity of @p hash, in bytes.
     @return if @p hash_size is positive, return the number of bytes written to @p hash.
     if @p hash_size is zero or negative, return the length of digest in bytes without changing @p hash.
     */
    int sha256(char* buffer, int size, char* hash, int hash_size);

    /**
     Print a string.
     @param[in] s the string to be printed.
     @param[in] l the length of @p s.
     */
    void print_str(char*s, int l);

    /**
     Print a signed 64-bit integer.
     @param[in] n the integer.
     */
    void print_int(long long n);

    /**
     Print an unsigned 64-bit integer.
     @param[in] n the integer.
     */
    void print_uint(unsigned long long n);

    /**
     Assert that contract has authority of specific account.
     @param[in] name the account name string.
     @param[in] length the length of @p name.
     @remarks
     This function aborts execution of contract if the authority check fails.
     */
    void require_auth(char* name, int length);

    /**
     Get balance of specific account.
     @param[in] name the account name string.
     @param[in] length the length of @p name.
     @return the balance of account @p name, in coins. If the account doesn't exist, abort execution.
     */
    unsigned long long get_user_balance(char* name, int length);

    /**
     Get balance of specific contract.
     @param[in] owner the account name who owns the contract.
     @param[in] owner_len the length of @p owner.
     @param[in] contract the name of the contract.
     @param[in] contract_len the length of @p contract.
     @return the balance of the contract, in coins. If the contract doesn't exist, abort execution.
     */
    unsigned long long get_contract_balance(char* owner, int owner_len, char* contract, int contract_len);

    /**
     Query a record in a database table.
     @param[in] table_name name of the table.
     @param[in] table_name_len length of @p table_name.
     @param[in] primary the primary key for query.
     @param[in] primary_len length of @p primary.
     @param[in,out] value the buffer to which record data are stored.
     @param[in] value_len capacity of @p value, in bytes
     @return if @p value_len is positive, return the number of bytes written in @p value.
     if @p value_len is zero or negative, return the size of value.
     */
    int table_get_record(char *table_name, int table_name_len, char* primary, int primary_len, char* value, int value_len);

    /**
     Create a record in a database table.
     @param[in] table_name name of the table.
     @param[in] table_name_len length of @p table_name.
     @param[in] value the record value.
     @param[in] value_len length of @p value.
     */
    void table_new_record(char *table_name, int table_name_len, char* value, int value_len);

    /**
     Update a record in a database table.
     @param[in] table_name name of the table.
     @param[in] table_name_len length of @p table_name.
     @param[in] primary the primary key of the record.
     @param[in] primary_len length of @p primary.
     @param[in] value the updated record value.
     @param[in] value_len length of @p value.
     */
    void table_update_record(char *table_name, int table_name_len, char* primary, int primary_len, char* value, int value_len);

    /**
     Delete a record in a database table.
     @param[in] table_name name of the table.
     @param[in] table_name_len length of @p table_name.
     @param[in] primary the primary key of the record.
     @param[in] primary_len length of @p primary.
     */
    void table_delete_record(char *table_name, int table_name_len, char* primary, int primary_len);

    /**
     Assert function
     @param[in] pred a boolean predicate.
     @param[in] msg the error message string.
     @param[in] msg_len the length of @p msg.
     @remarks
     This function aborts execution of contract if @p pred is zero. Otherwise, do nothing.
     */
    void cos_assert(int pred, char* msg, int msg_len);

    /**
     Abort execution of contract.
     */
    void abort();

    /**
     Get parameters data of current contract.
     @param[in,out] buf the buffer to which parameter data are stored.
     @param[in] size capacity of @p buf.
     @return if @p size is positive, return the number of bytes written to @p buf.
     if @p size is zero or negative, return the actual length of parameter data without changing @p buf.
     */
    int read_contract_op_params(char* buf, int size);

    /**
     Get amount of coins the caller has sent with current contract calling.
     @return the amount of coins.
     */
    unsigned long long read_contract_sender_value();

    /**
     Get name of current contract.
     @param[in,out] buf the buffer to which name is stored.
     @param[in] size capacity of @p buf, in bytes.
     @return if @p size is positive, return the number of bytes written to @p buf.
     if @p size is zero or negative, return the length of contract name.
     */
    int read_contract_name(char* buf, int size);

    /**
     Get name of current contract method.
     @param[in,out] buf the buffer to which name is stored.
     @param[in] size capacity of @p buf, in bytes.
     @return if @p size is positive, return the number of bytes written to @p buf.
     if @p size is zero or negative, return the length of method name.
     */
    int read_contract_method(char* buf, int size);

    /**
     Get name of the account who owns current contract.
     @param[in,out] buf the buffer to which name is stored.
     @param[in] size capacity of @p buf, in bytes.
     @return if @p size is positive, return the number of bytes written to @p buf.
     if @p size is zero or negative, return the length of owner account name.
     */
    int read_contract_owner(char* buf, int size);

    /**
     Get name of the account who is calling current contract.
     @param[in,out] buf the buffer to which name is stored.
     @param[in] size capacity of @p buf, in bytes.
     @return if @p size is positive, return the number of bytes written to @p buf.
     if @p size is zero or negative, return the length of calling account name.
     */
    int read_contract_caller(char* buf, int size);

    /**
     Return whether the contract was called directly by a user.
     @return 1 if called directly by a user, or 1 if called by a contract.
     */
    int contract_called_by_user();

    /**
     Get the owner account name of the calling contract.
     @param[in,out] buf the buffer to which name is stored.
     @param[in] size capacity of @p buf, in bytes.
     @return if @p size is positive, return the number of bytes written to @p buf.
     if @p size is zero or negative, return the length of owner account name.
     if current contract was directly called by a user, returns 0 without writing anything to @p buf.
     */
    int read_calling_contract_owner(char *buf, int size);

    /**
     Get the name of the calling contract.
     @param[in,out] buf the buffer to which name is stored.
     @param[in] size capacity of @p buf, in bytes.
     @return if @p size is positive, return the number of bytes written to @p buf.
     if @p size is zero or negative, return the length of calling contract name.
     */
    int read_calling_contract_name(char *buf, int size);

    /**
     Call other contract.
     @param[in] owner the owner account name of target contract.
     @param[in] owner_size length of @p owner.
     @param[in] contract the name of target contract.
     @param[in] contract_size length of @p contract.
     @param[in] method the name of target method.
     @param[in] method_size length of @p method.
     @param[in] params the packed parameters data.
     @param[in] params_size length of @p params.
     @param[in] coins amount of coins to be sent to the target contract.
     */
    void contract_call(char *owner, int owner_size, char *contract, int contract_size, char *method, int method_size, char *params, int params_size, unsigned long long coins);

    /**
     Transfer coins to specified user.
     @param[in] to account name of receiver.
     @param[in] to_len length of @p to.
     @param[in] amount number of coins to transfer.
     @param[in] memo a memo string.
     @param[in] memo_len length of @p memo.
     */
    void transfer_to_user( char* to, int to_len, unsigned long long amount, char* memo, int memo_len);

    /**
     Transfer coins to specified contract.
     @param[in] to_owner owner account name of receiver contract.
     @param[in] to_owner_len length of @p to_owner.
     @param[in] to_contract name of receiver contract.
     @param[in] to_contract_len length of @p to.
     @param[in] amount number of coins to transfer.
     @param[in] memo a memo string.
     @param[in] memo_len length of @p memo.
     */
    void transfer_to_contract( char* to_owner, int to_owner_len, char* to_contract, int to_contract_len, unsigned long long amount, char* memo, int memo_len);

#ifdef __cplusplus
}
#endif
