extern "C" {
    void print_str(char*, int);
    void print_uint(long long);

    int main() {
        char in[] = "hello world\n";
        print_str(in, 11);
        print_uint(42);
        return 0;
    }
}