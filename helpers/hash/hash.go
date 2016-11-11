package hash

/*
unsigned int hash_wt6(const char *key, int len)
{
	unsigned h0 = 0xa53c965aUL;
	unsigned h1 = 0x5ca6953aUL;
	unsigned step0 = 6;
	unsigned step1 = 18;

	for (; len > 0; len--) {
		unsigned int t;

		t = ((unsigned int)*key);
		key++;

		h0 = ~(h0 ^ t);
		h1 = ~(h1 + t);

		t  = (h1 << step0) | (h1 >> (32-step0));
		h1 = (h0 << step1) | (h0 >> (32-step1));
		h0 = t;

		t = ((h0 >> 16) ^ h1) & 0xffff;
		step0 = t & 0x1F;
		step1 = t >> 11;
	}
	return h0 ^ h1;
}

unsigned int hash_djb2(const char *key, int len)
{
	unsigned int hash = 5381;

	for (; len >= 8; len -= 8) {
		hash = ((hash << 5) + hash) + *key++;
		hash = ((hash << 5) + hash) + *key++;
		hash = ((hash << 5) + hash) + *key++;
		hash = ((hash << 5) + hash) + *key++;
		hash = ((hash << 5) + hash) + *key++;
		hash = ((hash << 5) + hash) + *key++;
		hash = ((hash << 5) + hash) + *key++;
		hash = ((hash << 5) + hash) + *key++;
	}
	switch (len) {
	case 7: hash = ((hash << 5) + hash) + *key++; 
	case 6: hash = ((hash << 5) + hash) + *key++; 
	case 5: hash = ((hash << 5) + hash) + *key++; 
	case 4: hash = ((hash << 5) + hash) + *key++; 
	case 3: hash = ((hash << 5) + hash) + *key++; 
	case 2: hash = ((hash << 5) + hash) + *key++; 
	case 1: hash = ((hash << 5) + hash) + *key++; break;
	default:  break;
	}
	return hash;
}

unsigned int hash_sdbm(const char *key, int len)
{
	unsigned int hash = 0;
	int c;

	while (len--) {
		c = *key++;
		hash = c + (hash << 6) + (hash << 16) - hash;
	}

	return hash;
}

unsigned int hash_crc32(const char *key, int len)
{
	unsigned int hash;
	int bit;

	hash = ~0;
	while (len--) {
		hash ^= *key++;
		for (bit = 0; bit < 8; bit++)
			hash = (hash >> 1) ^ ((hash & 1) ? 0xedb88320 : 0);
	}
	return ~hash;
}

 */
import (
	"C"
)

func CRC32(input string) uint32 {
	return uint32(C.hash_crc32(C.CString(input), C.int(len(input))))
}

func SDBM(input string) uint32 {
	return uint32(C.hash_sdbm(C.CString(input), C.int(len(input))))
}

func DJB2(input string) uint32 {
	return uint32(C.hash_djb2(C.CString(input), C.int(len(input))))
}

func WT6(input string) uint32 {
	return uint32(C.hash_wt6(C.CString(input), C.int(len(input))))
}
