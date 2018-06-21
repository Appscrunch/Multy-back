package currencies

// Satoshi + WEI
const (
	Satoshi = int64(100000000)
	Wei     = int64(1000000000000000)
)

// Dividers for BTC and ETH
var Dividers = map[int]int64{
	Bitcoin: Satoshi,
	Ether:   Wei,
}
