extern "C" {
    int readt2(char*);

    int main() {
        char in[] = "hello world";
        int ret = readt2(in);
        return ret;
    }
}