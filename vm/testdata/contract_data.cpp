extern "C" {
    long long get_balance_by_name(char*, int);
    long long get_contract_balance(char*, int, char*, int);
    void read_contract_owner(char*, int);
    void read_contract_caller(char*, int);
    long long get_sender_value();
    void print_uint64(long long);
    void print_string(char*, int);

    int main() {
        char name[] = "initminer";
        long long balance = get_balance_by_name(name, 9);
        print_uint64(balance);
//        char contract_owner[] = "initminer";
//        char cname[] = "hello";
//        long long contract_balance = get_contract_balance(contract_owner, 9, cname, 5);
//        print_uint64(balance);
//        char owner[20];
//        char caller[20];
//        read_contract_owner(owner, 20);
//        print_string(owner, 20);
//        read_contract_caller(caller, 20);
//        print_string(caller, 20);
//        long long sender_amount = get_sender_value();
//        print_uint64(sender_amount);
        return 0;
    }
}