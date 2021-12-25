package simplegocoin

import "time"

type Block struct {
	timeStamp     time.Time
	transaction   []Transaction
	previousBlock string
}
