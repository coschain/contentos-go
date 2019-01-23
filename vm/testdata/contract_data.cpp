extern "C" {
    long long get_user_balance(char*, int);
    long long get_contract_balance(char*, int, char*, int);
    void read_contract_owner(char*, int);
    void read_contract_caller(char*, int);
    long long read_contract_sender_value();
    void print_uint(long long);
    void print_str(char*, int);

    int main() {
        char name[] = "initminer";
        long long balance = get_user_balance(name, 9);
        print_uint(balance);
        char contract_owner[] = "initminer";
        char cname[] = "hello";
        long long contract_balance = get_contract_balance(cname, 5, contract_owner, 9);
        print_uint(contract_balance);
        char owner[20];
        char caller[20];
        read_contract_owner(owner, 20);
        print_str(owner, 20);
        read_contract_caller(caller, 20);
        print_str(caller, 20);
        long long sender_amount = read_contract_sender_value();
        print_uint(sender_amount);
        return 0;
    }
}