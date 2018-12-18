extern "C" {
    void sha256(char*, int, char*, int);
    void print_string(char*, int);

    int main() {
        char in[] = "hello world";
        int length = 11;
        char out[32];
        sha256(in, length, out, 32);
        print_string(out, 32);
        return 0;
    }
}