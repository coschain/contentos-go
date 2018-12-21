extern "C" {
    void print_string(char*, int);
    void print_uint32(int);
    void print_uint64(long long);
    void print_bool(int);

    int main() {
        char in[] = "hello world\n";
        print_string(in, 11);
        print_uint32(42);
        print_uint64((long long)1000);
        print_bool(1);
        print_bool(0);
        return 0;
    }
}