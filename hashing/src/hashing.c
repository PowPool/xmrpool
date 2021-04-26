#include "crypto/hash-ops.h"

void cryptonight_hash(const char* input, char* output, uint32_t len, uint64_t height) {
    const int variant = input[0] >= 7 ? input[0] - 6 : 0;
    cn_slow_hash(input, len, output, variant, 0, height);
}

void cryptonight_fast_hash(const char* input, char* output, uint32_t len) {
    cn_fast_hash(input, len, output);
}

uint64_t randomx_seedheight(uint64_t mainheight) {
    return rx_seedheight(mainheight);
}

void randomx_slow_hash(const uint64_t mainheight, const uint64_t seedheight,
    const char *seedhash, const char *data, uint32_t length,
    char *hash, uint32_t miners, uint32_t is_alt) {
    rx_slow_hash(mainheight, seedheight, seedhash, data, length, hash, miners, is_alt);
}