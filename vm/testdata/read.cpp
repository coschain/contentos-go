extern "C" {
    int readt1(char*, int);

    int main() {
        char in[] = "hello world";
        int length = 11;
        int ret = readt1(in, length);
        return ret;
    }
}