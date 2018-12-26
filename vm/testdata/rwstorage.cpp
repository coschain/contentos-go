extern "C" {
    void save_to_storage(char*, int, char*, int);
    void read_from_storage(char*, int, char*, int);
    void print_str(char*, int);

    int main() {
        char key[] = "hello";
        char value[] = "world";
        char rvalue[5];
        save_to_storage(key, 5, value, 5);
        read_from_storage(key, 5, rvalue, 5);
        print_str(rvalue, 5);

        return 0;
    }
}