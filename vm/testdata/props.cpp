extern "C" {
    long long current_block_number();
    long long current_timestamp();
    int current_witness(char *);
    void print_uint64(long long);
    void print_uint32(int);
    void print_string(char*, int);

    int main() {
        print_uint64(current_block_number());
        print_uint64(current_timestamp());
        char witness[32];
        int length = current_witness(witness);
        print_uint32(length);
        print_string(witness, length);
        return 0;
    }
}