extern "C" {

    float a = 3.14;

    int sub(int b) {
        return  int(a)-b;
    }

    int main() {
        float c = sub(1);
        return 0;
    }
}