extern "C" {
    int copy(char*, char*, int);
    void print_str(char*, int);
    void print_uint(int);

    int main() {
        char in[] = "hello world";
        char out[15];
        int ret = copy(in, out, 11);
        print_str(out, 11);
        print_uint(ret);
        return 0;
    }
}