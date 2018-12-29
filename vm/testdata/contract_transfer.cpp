extern "C" {
    long long get_balance_by_name(char*, int);
    void transfer(char*, int, long long, char*, int);
    long long get_contract_balance(char*, int, char*, int);
    void print_uint(long long);

    int main() {
        char name[] = "alice";
        long long balance = get_balance_by_name(name, 5);
        print_uint(balance);
        char contract_owner[] = "initminer";
        char cname[] = "hello";
        long long contract_balance = get_contract_balance(cname, 5, contract_owner, 9);
        print_uint(contract_balance);
        char caller[] = "alice";

        char memo[] = "hello";

        transfer(caller, 5, 1000, memo, 5);

        balance = get_balance_by_name(caller, 5);
        print_uint(balance);
        contract_balance = get_contract_balance(cname, 5, contract_owner, 9);
        print_uint(contract_balance);
        return 0;
    }
}