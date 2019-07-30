#include <cosiolib/contract.hpp>
#include <cosiolib/print.hpp>


class constraints : public cosio::contract {
public:
    using cosio::contract::contract;

    void alloc_mem(uint32_t size) {
        cosio::checksum256 hash;
        char *buf = new char[size];
        memset(buf, 0xaa, size);
        ::sha256(buf, size, (char*)hash.hash, 32);
        delete []buf;
        cosio::print(hash.to_string());
    }

    int32_t recursive(int32_t level) {
        if (level < 3) {
            return level;
        }
        if (cosio::current_timestamp() % 2) {
            return 2 + recursive(level - 1);
        } else {
            return 5 + recursive(level - 1);
        }
    }

    void call_depth(int32_t depth) {
        auto r = recursive(depth);
        cosio::print(r);
    }

    void infinite_loop() {
        uint64_t n = 1, r = 0;
        while(n++) { 
            r ^= n & 0xffeebbaa;
            r <<= 1;
        }
        cosio::print(r);
    }
};

COSIO_ABI(constraints, (alloc_mem)(call_depth)(infinite_loop))
