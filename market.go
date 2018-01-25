package main

import (
	"fmt"
	"github.com/rkjdid/errors"
	"github.com/rkjdid/util"
	"github.com/toorop/go-bittrex"
	"log"
	"time"
)

type Market struct {
	bittrex.Market
	Config     *MarketConfig
	Candles    []bittrex.Candle
	Indicators []Indicator
	LastCandle bittrex.Candle
	Client     *bittrex.Bittrex

	stop chan interface{}
}

func NewMarket(market bittrex.Market, cfg MarketConfig, client *bittrex.Bittrex) *Market {
	return &Market{
		Market:     market,
		Config:     &cfg,
		Client:     client,
		Indicators: make([]Indicator, 0),
	}
}

func (m *Market) AddIndicator(ind Indicator) {
	m.Indicators = append(m.Indicators, ind)
}

func max(ints ...int) int {
	if len(ints) == 0 {
		return 0
	} else if len(ints) == 1 {
		return ints[0]
	}

	maxValue := ints[0]
	for _, i := range ints[1:] {
		if i > maxValue {
			maxValue = i
		}
	}
	return maxValue
}

// CandleTimeDiff performs a time comparison between c0 and c1 by calling time.Sub on both timestamps.
// If returned duration is negative, c0 is before c1
//                      is positive, c0 is after c1
//                      equals 0, c0 and c1 represent the same time.
func CandleTimeDiff(c0, c1 bittrex.Candle) time.Duration {
	return c0.TimeStamp.Time.Sub(c1.TimeStamp.Time)
}

func (m *Market) IsCandleNew(candle bittrex.Candle) bool {
	return CandleTimeDiff(candle, m.LastCandle) > 0
}

func (m *Market) GetLatestTick() (cdl bittrex.Candle, err error) {
	candles, err := m.Client.GetLatestTick(m.MarketName, string(m.Config.Interval))
	if err != nil {
		return cdl, err
	}
	return candles[0], nil
}

// FillCandles retreives the last sz candles from market history
// and fills moving averages so next candle works with relevant data.
func (m *Market) PrefillCandles() error {
	candles, err := m.Client.GetTicks(m.MarketName, string(m.Config.Interval))
	if err != nil {
		return err
	}

	sz := m.Config.Prefill
	m.Candles = make([]bittrex.Candle, sz)
	if sz > len(candles) {
		sz = len(candles)
	}

	for i := len(candles) - sz; i < len(candles); i++ {
		m.AddTick(candles[i], true)
	}
	return nil
}

// AddCandle inserts c as the last candle, recomputing Values()
// and returning true if added candle was a hit.
func (m *Market) AddTick(c bittrex.Candle, fillOnly bool) (errs error) {
	m.LastCandle = c
	for _, v := range m.Indicators {
		err = v.AddTick(c)
		if err != nil {
			log.Printf("%s: %s", v, err)
			errs = errors.Add(errs, err)
		}
	}
	return errs
}

func (m *Market) StartPolling() {
	var (
		// poll for a bit less than duration, so we can try and catch up delay
		// without polling to much unnecessarily
		longPoll = time.Duration(Candles[m.Config.Interval]) * (4 / 5)
		// become somewhat aggressive towards the end of the interval
		shortPoll = time.Duration(Candles[m.Config.Interval]) / 30
		name      = m.MarketName
		timer     = time.NewTimer(shortPoll)
		c         bittrex.Candle
	)

	m.stop = make(chan interface{})
	for {
		candles, err := m.Client.GetLatestTick(name, string(m.Config.Interval))
		if err != nil {
			log.Printf("bittrex GetLatestTick %s: %s", name, err)
			timer.Reset(shortPoll)
			goto sleep
		}
		c = candles[0]
		if !m.IsCandleNew(c) {
			m.LastCandle = c
			timer.Reset(shortPoll)
			goto sleep
		}

		// We have a new candle, we insert previous m.LastCandle for computation,
		// which is recent enough hopefully.
		_ = m.AddTick(m.LastCandle, false)

		// store new candle
		m.LastCandle = c

		// we have new value, long poll
		timer.Reset(longPoll)

	sleep:
		select {
		case <-m.stop:
			return
		case <-timer.C:
			// default to short poll
			timer.Reset(shortPoll)
		}
	}
}

func (m *Market) Stop() {
	if m.stop == nil {
		return
	}
	m.stop <- nil
	m.stop = nil
}

func (m *Market) String() string {
	return fmt.Sprintf("%s@%s", m.MarketName, util.ParisTime(m.LastCandle.TimeStamp.Time).Format("2006-01-02 15:04:05 MST"))
}
