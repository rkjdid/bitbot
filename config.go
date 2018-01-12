package main

import (
	"github.com/rkjdid/util"
	"time"
)

type CandleInterval string

const (
	CandleMinute    = CandleInterval("oneMin")
	Candle5Minutes  = CandleInterval("fiveMin")
	Candle30Minutes = CandleInterval("thirtyMin")
	CandleHour      = CandleInterval("hour")
	CandleDay       = CandleInterval("day")
)

var Candles = map[CandleInterval]util.Duration{
	CandleMinute:    util.Duration(time.Minute),
	Candle5Minutes:  util.Duration(time.Minute * 5),
	Candle30Minutes: util.Duration(time.Minute * 30),
	CandleHour:      util.Duration(time.Hour),
	CandleDay:       util.Duration(time.Hour * 24),
}

type Config struct {
	Scanner *ScannerConfig
}

type ScannerConfig struct {
	// only pick those markets, if nil or empty all BTC markets are monitored
	Pairs []string

	// only pick markets whose daily volume is greater than
	MinBtcVolumeDaily float64

	// candle time string: one of "oneMin" "fiveMin" "thirtyMin" "hour" "day"
	Interval CandleInterval

	// long term MA length
	LongTerm int

	// short term MA length
	ShortTerm int

	BBLength   int
	Multiplier float64

	// number of consecutive hits to trigger notification
	NotificationThreshold int

	BittrexApiKey    string
	BittrexApiSecret string
}

var DefaultConfig = Config{
	Scanner: &ScannerConfig{
		Pairs:                 []string{},
		Interval:              Candle30Minutes,
		LongTerm:              20,
		ShortTerm:             5,
		BBLength:              20,
		Multiplier:            2.5,
		MinBtcVolumeDaily:     200.0,
		NotificationThreshold: 0,
	},
}

// IsValid checks that we're working with a meaningful config, preventing from
// using malformed config & potentially panic-ing around.
func (cfg Config) IsValid() bool {
	return cfg.Scanner.BBLength > 0 && cfg.Scanner.LongTerm > cfg.Scanner.ShortTerm && cfg.Scanner.ShortTerm > 0
}
