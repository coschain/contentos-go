extern "C" {
    void sha256(char*, int, char*, int);

    int add (int a, int b) {
        int c = 10;
        int d = c + 19;
        return a + b + d;
    }

    int main() {
        int c = add(1, 2);
        char in[] = "hello world";
        int length = 11;
        char out[32];
        sha256(in, length, out, 32);
        return 0;
    }
}