extern "C" {
    void sha256(char*, int, char*, int);
    void print_str(char*, int);

    int main() {
        char in[] = "hello world";
        int length = 11;
        char out[32];
        sha256(in, length, out, 32);
        print_str(out, 32);
        return 0;
    }
}