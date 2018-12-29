extern "C" {
    long long current_block_number();
    long long current_timestamp();
    int current_witness(char *, int);
    void print_uint(int);
    void print_str(char*, int);

    int main() {
        print_uint(current_block_number());
        print_uint(current_timestamp());
        char witness[32];
        int length = current_witness(witness, 0);
        current_witness(witness, length);
        print_uint(length);
        print_str(witness, length);
        return 0;
    }
}