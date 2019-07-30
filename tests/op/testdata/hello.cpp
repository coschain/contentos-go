#include <cosiolib/contract.hpp>
#include <cosiolib/print.hpp>


class hello : public cosio::contract {
public:
    using cosio::contract::contract;

    void greet() {
        cosio::print(cosio::get_contract_caller());
    }
};

COSIO_ABI(hello, (greet))
