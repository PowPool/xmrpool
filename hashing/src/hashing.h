void cryptonight_hash(const char* input, char* output, uint32_t len, uint64_t height);
void cryptonight_fast_hash(const char* input, char* output, uint32_t len);
uint64_t randomx_seedheight(uint64_t mainheight);
void randomx_slow_hash(const uint64_t mainheight, const uint64_t seedheight, const char *seedhash, const void *data, size_t length, char *hash, int miners, int is_alt);
