extern "C" {
    int readt2(char*);

    int main() {
        char in[] = "hello\0 world";
        int ret = readt2(in);
        return ret;
    }
}