#include <cosiolib/contract.hpp>
#include <cosiolib/print.hpp>

static uint32_t current_block_number(cosio::contract *c) {
    return cosio::current_block_number();
}

static uint32_t current_timestamp(cosio::contract *c) {
    return cosio::current_timestamp();
}

static uint32_t current_witness(cosio::contract *c) {
    return cosio::current_witness().size();
}

static uint32_t block_producers(cosio::contract *c) {
    uint32_t r = 0;
    auto producers = cosio::block_producers();
    for (auto it = producers.begin(); it != producers.end(); it++) {
        r += it->size();
    }
    return r;
}

static uint32_t sha256(cosio::contract *c) {
    const size_t size = 16 * 1024;
    uint32_t r = 0;
    char hash[32];
    char *data = new char[size];
    memset(data, 0xab, size);
    ::sha256(data, size, hash, 32);
    r = *(uint32_t*)hash;
    delete []data;
    return r;
}

static uint32_t is_contract_called_by_user(cosio::contract *c) {
    return cosio::is_contract_called_by_user();
}

static uint32_t get_contract_caller(cosio::contract *c) {
    return c->get_caller().string().size();
}

static uint32_t get_contract_name(cosio::contract *c) {
    return c->get_name().string().size();
}

static uint32_t get_contract_method(cosio::contract *c) {
    return cosio::get_contract_method().size();
}

static uint32_t get_contract_sender_value(cosio::contract *c) {
    return cosio::get_contract_sender_value();
}

static uint32_t get_contract_balance(cosio::contract *c) {
    return cosio::get_contract_balance(c->get_name());
}

static uint32_t get_user_balance(cosio::contract *c) {
    return cosio::get_user_balance(c->get_name().account());
}

static uint32_t require_auth(cosio::contract *c) {
    cosio::require_auth(c->get_caller());
    return 0xaabbccdd;
}

static uint32_t transfer(cosio::contract *c) {
    cosio::transfer_to(c->get_name(), 0, "");
    return 0x11442278;
}

static uint32_t print(cosio::contract *c) {
    cosio::print(c->get_name(), 12345);
    cosio::print_f("hello % % %", c->get_name(), 56789, "world");
    return 0x734933ef;
}

static uint32_t int_ops(cosio::contract *c) {
    uint32_t r = c->get_name().string().size();
    for (int i = 0; i < 16; i++) {
        int s = r & 7;
        switch(s) {
            case 0: r ^= 0x12345678; break;
            case 1: r ^= 0x23456781; break;
            case 2: r ^= 0x34567812; break;
            case 3: r ^= 0x45678123; break;
            case 4: r ^= 0x56781234; break;
            case 5: r ^= 0x67812345; break;
            case 6: r ^= 0x78123456; break;
        }
        r <<= 3;
        r += 0x232323;
        r *= 121061;
        r -= 280769;
        r ^= 0xffeced00;
    }
    return r;
}


uint32_t (*g_burners[])(cosio::contract *) = {
    current_block_number,
    current_timestamp,
    current_witness,
    block_producers,
    sha256,
    is_contract_called_by_user,
    get_contract_caller,
    get_contract_name,
    get_contract_method, 
    get_contract_sender_value,
    get_contract_balance,
    get_user_balance,
    require_auth,
    transfer,
    print,
    int_ops,
};

size_t g_burners_count = sizeof(g_burners) / sizeof(uint32_t(*)(cosio::contract *));

