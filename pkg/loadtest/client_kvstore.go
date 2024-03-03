package loadtest

import (
	"fmt"
)

// The Tendermint common.RandStr method can effectively generate human-readable
// (alphanumeric) strings from a set of 62 characters. We aim here with the
// KVStore client to generate unique client IDs as well as totally unique keys
// for all transactions. Values are not so important.
const (
	KVStoreClientIDLen int = 5 // Allows for 6,471,002 random client IDs (62C5)允许生成 6,471,002 个随机客户端ID（62的5次方）
	kvstoreMinValueLen int = 1 // We at least need 1 character in a key/value pair's value.键/值对的值至少需要1个字符。
)

// This is a map of nCr where n=62 and r varies from 0 through 15. It gives the
// maximum number of unique transaction IDs that can be accommodated with a
// given key suffix length.
/*在 KVStoreClient 结构中，键的生成是通过拼接客户端ID和随机生成的键后缀来实现的。为了确保生成的键在一组事务中是唯一的，需要计算在给定键后缀长度的情况下，可以容纳的最大唯一交易ID数量。这样可以在生成键时使用足够长的后缀，以确保在整个测试中不会出现键冲突。
具体而言，kvstoreMaxTxsByKeySuffixLen 数组的每个元素表示在给定键后缀长度的情况下，可以容纳的最大唯一交易ID数量。在 requiredKVStoreSuffixLen 函数中，通过比较最大交易数与数组中的值，找到适当的后缀长度，以确保可以容纳足够的唯一交易ID数量。这有助于在生成随机键时，减少键冲突的可能性，从而更好地模拟实际负载情况。
这个函数的目的是根据最大交易数找到适当的键后缀长度。生成的键后缀长度就能够保证在整个测试中不会出现键冲突。
*/
var kvstoreMaxTxsByKeySuffixLen = []uint64{ //一个数组，表示在给定键后缀长度的情况下，可以容纳的最大唯一交易ID数量。
	0,              // 0
	62,             // 1
	1891,           // 2
	37820,          // 3
	557845,         // 4
	6471002,        // 5
	61474519,       // 6
	491796152,      // 7
	3381098545,     // 8
	20286591270,    // 9
	107518933731,   // 10
	508271323092,   // 11
	2160153123141,  // 12
	8308281242850,  // 13
	29078984349975, // 14
	93052749919920, // 15
}

// KVStoreClientFactory creates load testing clients to interact with the
// built-in Tendermint kvstore ABCI application.
type KVStoreClientFactory struct{}

// KVStoreClient 结构生成任意交易（随机键/值对），将其发送到kvstore ABCI应用程序。
// KVStoreClient generates arbitrary transactions (random key=value pairs) to
// be sent to the kvstore ABCI application. The keys are structured as follows:
//
// `[client_id][tx_id]=[tx_id]`
//
// where each value (`client_id` and `tx_id`) is padded with 0s to meet the
// transaction size requirement.
// 都使用0填充，以满足交易大小的要求
type KVStoreClient struct {
	keyPrefix    []byte // Contains the client ID
	keySuffixLen int
	valueLen     int
}

var (
	_ ClientFactory = (*KVStoreClientFactory)(nil)
	_ Client        = (*KVStoreClient)(nil)
)

// 初始化函数，在包被导入时注册KVStoreClientFactory
func init() {
	if err := RegisterClientFactory("kvstore", NewKVStoreClientFactory()); err != nil {
		panic(err)
	}
}

// NewKVStoreClientFactory 函数返回一个新的KVStoreClientFactory实例
func NewKVStoreClientFactory() *KVStoreClientFactory {
	return &KVStoreClientFactory{}
}

// ValidateConfig 方法用于验证配置是否合法
func (f *KVStoreClientFactory) ValidateConfig(cfg Config) error {
	maxTxsPerEndpoint := cfg.MaxTxsPerEndpoint()
	if maxTxsPerEndpoint < 1 {
		return fmt.Errorf("cannot calculate an appropriate maximum number of transactions per endpoint (got %d)", maxTxsPerEndpoint)
	}
	minKeySuffixLen, err := requiredKVStoreSuffixLen(maxTxsPerEndpoint)
	if err != nil {
		return err
	}
	// "[client_id][random_suffix]=[value]"
	minTxSize := KVStoreClientIDLen + minKeySuffixLen + 1 + kvstoreMinValueLen
	if cfg.Size < minTxSize {
		return fmt.Errorf("transaction size %d is too small for given parameters (should be at least %d bytes)", cfg.Size, minTxSize)
	}
	return nil
}

// NewClient 方法创建一个新的KVStoreClient实例
func (f *KVStoreClientFactory) NewClient(cfg Config) (Client, error) {
	keyPrefix := []byte(randStr(KVStoreClientIDLen))
	keySuffixLen, err := requiredKVStoreSuffixLen(cfg.MaxTxsPerEndpoint())
	if err != nil {
		return nil, err
	}
	keyLen := len(keyPrefix) + keySuffixLen
	// value length = key length - 1 (to cater for "=" symbol)
	valueLen := cfg.Size - keyLen - 1
	return &KVStoreClient{
		keyPrefix:    keyPrefix,
		keySuffixLen: keySuffixLen,
		valueLen:     valueLen,
	}, nil
}

// requiredKVStoreSuffixLen 方法返回满足最大交易数要求的键后缀长度
func requiredKVStoreSuffixLen(maxTxCount uint64) (int, error) {
	for l, maxTxs := range kvstoreMaxTxsByKeySuffixLen {
		if maxTxCount < maxTxs {
			if l+1 > len(kvstoreMaxTxsByKeySuffixLen) {
				return -1, fmt.Errorf("cannot cater for maximum tx count of %d (too many unique transactions, suffix length %d)", maxTxCount, l+1)
			}
			// we use l+1 to minimize collision probability
			return l + 1, nil
		}
	}
	return -1, fmt.Errorf("cannot cater for maximum tx count of %d (too many unique transactions)", maxTxCount)
}

// GenerateTx 方法生成一个随机事务
func (c *KVStoreClient) GenerateTx() ([]byte, error) {
	k := append(c.keyPrefix, []byte(randStr(c.keySuffixLen))...)
	v := []byte(randStr(c.valueLen))
	return append(k, append([]byte("="), v...)...), nil
}
