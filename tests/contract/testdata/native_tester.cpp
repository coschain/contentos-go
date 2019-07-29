#include <cosiolib/contract.hpp>

class native_tester : public cosio::contract {
public:
    using cosio::contract::contract;

    void current_block_number(uint64_t expected) {
        cosio::cosio_assert(cosio::current_block_number() == expected, "current_block_number");
    }

    void current_timestamp(uint64_t expected) {
        cosio::cosio_assert(cosio::current_timestamp() == expected, "current_timestamp");
    }

    void current_witness(const std::string& expected) {
        cosio::cosio_assert(cosio::current_witness() == expected, "current_witness");
    }

    void block_producers(const std::vector<std::string>& expected) {
        cosio::cosio_assert(cosio::block_producers() == expected, "block_producers");
    }

    void sha256(const cosio::bytes& data, const cosio::checksum256& expected) {
        cosio::cosio_assert(memcmp(cosio::sha256(data).hash, expected.hash, 32) == 0, "sha256");
    }

    void is_contract_called_by_user(bool expected) {
        cosio::cosio_assert(cosio::is_contract_called_by_user() == expected, "is_contract_called_by_user");
    }

    void get_contract_caller(const std::string& expected) {
        cosio::cosio_assert(cosio::get_contract_caller() == expected, "get_contract_caller");
    }

    void get_contract_caller_contract(const std::string& owner, const std::string& name) {
        cosio::cosio_assert(cosio::_read_string(::read_calling_contract_owner) == owner && cosio::_read_string(::read_calling_contract_name) == name, "get_contract_caller_contract");
    }

    void get_contract_name(const std::string& owner, const std::string& name) {
        cosio::cosio_assert(cosio::get_contract_name() == cosio::name(owner, name), "get_contract_name");
    }

    void get_contract_method(const std::string& expected) {
        cosio::cosio_assert(cosio::get_contract_method() == expected, "get_contract_method");
    }

    void get_contract_sender_value(cosio::coin_amount expected) {
        cosio::cosio_assert(cosio::get_contract_sender_value() == expected, "get_contract_sender_value");
    }

    void get_contract_balance(const std::string& owner, const std::string& name, cosio::coin_amount expected) {
        cosio::cosio_assert(cosio::get_contract_balance(cosio::name(owner, name)) == expected, "get_contract_balance");
    }

    void get_user_balance(const std::string& name, cosio::coin_amount expected) {
        cosio::cosio_assert(cosio::get_user_balance(cosio::name(name)) == expected, "get_user_balance");
    }

    void require_auth(const std::string& name) {
        cosio::require_auth(name);
    }
};

COSIO_ABI(native_tester, (current_block_number)(current_timestamp)(current_witness)(block_producers)(sha256)(is_contract_called_by_user)(get_contract_caller)(get_contract_caller_contract)(get_contract_name)(get_contract_method)(get_contract_sender_value)(get_contract_balance)(get_user_balance)(require_auth))
