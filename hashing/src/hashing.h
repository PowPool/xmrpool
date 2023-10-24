void cryptonight_hash(const char* input, char* output, uint32_t len, uint64_t height);
void cryptonight_fast_hash(const char* input, char* output, uint32_t len);

//uint64_t randomx_seedheight(uint64_t mainheight);
//void randomx_slow_hash(const uint64_t mainheight, const uint64_t seedheight,
//    const char* seedhash, const char* data, uint32_t length,
//    char* hash, uint32_t miners, uint32_t is_alt);

void randomx_slow_hash(const char *seedhash, const char *data, uint32_t length, char *hash);