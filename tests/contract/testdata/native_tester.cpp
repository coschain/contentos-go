#include <cosiolib/contract.hpp>

class native_tester : public cosio::contract {
public:
    using cosio::contract::contract;

    void sha256(const cosio::bytes& data, const cosio::checksum256& expected) {
        cosio::cosio_assert(memcmp(cosio::sha256(data).hash, expected.hash, 32) == 0, "sha256 mismatch");
    }
};

COSIO_ABI(native_tester, (sha256))
