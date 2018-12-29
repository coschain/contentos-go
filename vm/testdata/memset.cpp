extern "C" {
    void cos_assert(int, char*, int);

    int main() {
        char in[100] = {2, 3};
        char msg[] = "hello";
        cos_assert(in[0] == int(2), msg, 20);
        cos_assert(in[1] == int(3), msg, 20);
        cos_assert(in[99] == int(0), msg, 20);
        return 0;
    }
}