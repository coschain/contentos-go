#include <cosiolib/contract.hpp>
#include <cosiolib/print.hpp>

class has_float : public cosio::contract {
public:
    using cosio::contract::contract;

    void hello(uint32_t n) {
        double d = 100.0;
        d *= 3.1415926;
        d += n;
        d /= 2.71828;
        ::print_int( *(int32_t*)&d );
    }
};

COSIO_ABI(has_float, (hello))
