#include <cosiolib/contract.hpp>
#include <cosiolib/print.hpp>

extern uint32_t (*g_burners[])(cosio::contract *);
extern size_t g_burners_count;

class gas_burner : public cosio::contract {
public:
    using cosio::contract::contract;

    uint32_t rand(uint32_t x) {
        return x * 134775813 + 1;
    }

    void burn(uint32_t seed) {
        uint32_t result = 0;
        uint32_t r = seed;
        for (size_t i = 0; i < g_burners_count; i++) {
            r = this->rand(r);
            auto times = r % 10 + 1;
            auto burner = g_burners[i];
            if (burner && times) {
                for (auto j = 0; j < times; j++) {
                    result ^= burner(this);
                }
            }
        }
        cosio::print(result);
    }
};

COSIO_ABI(gas_burner, (burn))
