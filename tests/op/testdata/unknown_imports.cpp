#include <cosiolib/contract.hpp>

extern "C" void unknown_func(uint32_t);

class unknown_imports : public cosio::contract {
public:
    using cosio::contract::contract;

    void hello(uint32_t n) {
        unknown_func(n);
    }
};

COSIO_ABI(unknown_imports, (hello))
