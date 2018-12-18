extern "C" {
    void cos_assert(int, char*, int);
    void print_string(char*, int);

    int main() {
        char msg[] = "assert error";
        char msg3[] = "should not be printed";
        cos_assert(false, msg, 20);
        print_string(msg3, 30);
        return 0;
    }
}