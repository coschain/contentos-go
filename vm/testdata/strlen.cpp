extern "C" {
    int readt1(char*, int);

    int main() {
        char * in = "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000";
        int ret = readt1(in, 0);
        return ret;
    }
}