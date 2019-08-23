#include <cosiolib/contract.hpp>
#include <cosiolib/print.hpp>
#include <cosiolib/system.hpp>

using namespace std;

//
// Token contract maintains 2 database tables.
// One is "stats" table, which contains a single row recording properties of the token.
// The other is "balances" table in which each row records a token account and its balance.
//

/**
 * @brief record type of "balances" table.
 */
struct balance {
    cosio::name tokenOwner;         ///< name of account who owns the token
    uint64_t amount;                ///< balance of the account

    // specify the sequence of fields for serialization.
    COSIO_SERIALIZE(balance, (tokenOwner)(amount))
};

/**
 * @brief record type of "stats" table.
 */
struct stat : public cosio::singleton_record {
    string name;                    ///< name of the token
    string symbol;                  ///< symbol name of the token
    uint64_t total_supply;          ///< total number of tokens issued
    uint32_t decimals;              ///< number of digits after decimal point

    // specify the sequence of fields for serialization.
    COSIO_SERIALIZE_DERIVED(stat, cosio::singleton_record, (name)(symbol)(total_supply)(decimals))
};

/**
 * @brief the token contract class
 */
struct cosToken : public cosio::contract {
    using cosio::contract::contract;

    /**
     * @brief contract method to create a new type of token.
     * 
     * @param name          name of the token, e.g. "Native token of Contentos".
     * @param symbol        symbol name of the token, e.g. "COS".
     * @param total_supply  total number of tokens to issue.
     * @param decimals      number of digits after decimal point, e.g. with decimals==3, 12345 COS tokens represent as "12.345 COS".
     */
    void create(string name,string symbol, uint64_t total_supply, uint32_t decimals) {
        // make sure that only the contract owner can create her token
        cosio::require_auth(get_name().account());

        // create the stats record in database with default record member values.
        stats.get_or_create();
        // update the stats record
        stats.update([&](stat& s){
                s.name = name;
                s.symbol = symbol;
                s.total_supply = total_supply;
                s.decimals = decimals;
                });

        // the contract owner owns all tokens
        // record it into the balance table
        auto owner = get_name();
        balances.insert([&](balance& b){
            auto account_name = owner.account();
            b.tokenOwner.set_string(account_name);
            b.amount = total_supply;
        });

        // [optional] query and print balance of contract owner.
        auto b = balances.get(owner.account());
        cosio::print_f("user % has % tokens. \n", b.tokenOwner.string(), b.amount);
    }

    /**
     * @brief contract method to transfer tokens.
     * 
     * @param from      the account who sends tokens.
     * @param to        the account who receives tokens.
     * @param amount    number of tokens to transfer.
     */
    void transfer(cosio::name from,cosio::name to, uint64_t amount) {
        // we need authority of sender account.
        cosio::require_auth(from);

        // check if sender has any tokens.
        cosio::cosio_assert(balances.has(from), std::string("no balance:") + from.string());
        // check if sender has enough tokens.
        cosio::cosio_assert(balances.get(from).amount >= amount, std::string("balance not enough:") + from.string());
        // check integer overflow
        cosio::cosio_assert(balances.get_or_default(to).amount + amount > balances.get_or_default(to).amount, std::string("over flow"));

        // total balance of both sender and receiver
        auto previousBalances = balances.get_or_default(from).amount + balances.get_or_default(to).amount;

        // decrease sender's balance
        balances.update(from,[&](balance& b){
                    b.amount -= amount;
                    });
        // increase receiver's balance
        if(!balances.has(to)) {
            balances.insert([&](balance& b){
                        b.tokenOwner = to;
                        b.amount = amount;
                    });
        } else {
            balances.update(to,[&](balance& b){
                       b.amount += amount;
                    });
        }

        // make sure that total balance of both accounts not changed.
        cosio::cosio_assert(balances.get(from).amount + balances.get(to).amount == previousBalances, std::string("balance not equal after transfer"));
    }

    //
    // define a class member named "balances" representing a database table which,
    // - has name of "balances", the same as variable name,
    // - has a record type of balance
    // - takes balance::tokenOwner as primary key
    //
    COSIO_DEFINE_TABLE( balances, balance, (tokenOwner) );

    //
    // define a class data member named "stats" representing a singleton table which,
    // - has name of "stats"
    // - has a record type of stat
    //
    COSIO_DEFINE_NAMED_SINGLETON( stats, "stats", stat );
};

// declare the class and methods of contract.
COSIO_ABI(cosToken, (create)(transfer))
