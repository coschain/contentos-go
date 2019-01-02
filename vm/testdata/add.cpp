extern "C" {
    int add(int, int);

    int add2(int a, int b) {
        return a + b;
    }

    int main() {
        int c = add(1, 2);
        int d = add2(3, 4);
        return c + d;
    }
}