extern "C" {

    float add(float a, float b);

    int main() {
        float a = 3.14;
        float c = add(a, 1.2);
        return 0;
    }
}