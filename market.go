package main

import (
	"fmt"
	"github.com/montanaflynn/stats"
	"github.com/rkjdid/util"
	"github.com/toorop/go-bittrex"
	"log"
	"time"
)

type Market struct {
	bittrex.Market
	Interval   CandleInterval
	Candles    []bittrex.Candle
	LastCandle bittrex.Candle

	ShortMAs   *MATrio
	LongMAs    *MATrio
	BBSum      *MovingAverage
	Multiplier float64

	ConsecutiveHits int
	TotalHits       int

	Client *bittrex.Bittrex

	stop chan interface{}
}

func NewMarket(market bittrex.Market, longLength, shortLength, bbLength int,
	interval CandleInterval, multiplier float64, client *bittrex.Bittrex) *Market {

	return &Market{
		Market:     market,
		ShortMAs:   NewMATrio(shortLength),
		LongMAs:    NewMATrio(longLength),
		BBSum:      NewMovingAverage(bbLength),
		Interval:   interval,
		Client:     client,
		Multiplier: multiplier,
	}
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

func CandlesEqual(c0, c1 bittrex.Candle) bool {
	return c0.TimeStamp.Time.Equal(c1.TimeStamp.Time) && c0.Open.Equal(c1.Open)
}

func (m *Market) IsCandleNew(candle bittrex.Candle) bool {
	return !CandlesEqual(m.LastCandle, candle)
}

func (m *Market) GetLatestTick() (cdl bittrex.Candle, err error) {
	candles, err := m.Client.GetLatestTick(m.MarketName, string(m.Interval))
	if err != nil {
		return cdl, err
	}
	return candles[0], nil
}

// FillCandles retreives the last n candles from market history
// and fills moving averages so next candle works with relevant data.
func (m *Market) FillCandles() error {
	candles, err := m.Client.GetTicks(m.MarketName, string(m.Interval))
	if err != nil {
		return err
	}

	sz := max(m.LongMAs.Length, m.ShortMAs.Length, m.BBSum.Window)
	m.Candles = make([]bittrex.Candle, sz)

	if sz > len(candles) {
		sz = len(candles)
	}

	for i := len(candles) - sz; i < sz; i++ {
		m.AddCandle(candles[i], true)
	}
	return nil
}

// AddCandle inserts c as the last candle, recomputing values
// and returning true if added candle was a hit.
func (m *Market) AddCandle(c bittrex.Candle, fillOnly bool) bool {
	m.LastCandle = c
	p, _ := c.Close.Float64()
	v, _ := c.Volume.Float64()
	m.ShortMAs.Add(p, v)
	m.LongMAs.Add(p, v)

	vpc := m.LongMAs.PV.Avg()/m.LongMAs.V.Avg() - m.LongMAs.P.Avg()
	vpr := m.ShortMAs.PV.Avg() / m.ShortMAs.V.Avg() / m.ShortMAs.P.Avg()
	vm := m.ShortMAs.V.Avg() / m.LongMAs.V.Avg()
	vpci := vpc * vpr * vm

	m.BBSum.Add(vpci)
	if fillOnly {
		return false
	}

	dev, err := stats.StandardDeviation(stats.Float64Data(m.BBSum.Values))
	if err != nil {
		log.Panicf("stdev shouldnt error: %s (len: %d)", err, len(m.BBSum.Values))
	}

	basis := m.BBSum.Avg()
	dev *= m.Multiplier

	// hit detection
	hit := vpci > (basis + dev)

	if hit {
		m.ConsecutiveHits += 1
		m.TotalHits += 1
	} else {
		m.ConsecutiveHits = 0
	}

	bv, _ := c.BaseVolume.Float64()
	log.Printf("SPAM %15s - vpc: %8f, vpr: %8f, vm: %8f, vpci: %8f, basis: %8f, dev: %8f\n\t\t"+
		"candle: %s, price: %8f, btc_vol: %8f, MA_P: %8f / %8f, MA_V: %8f / %8f, MA_PV: %8f / %8f",
		m.MarketName, vpc, vpr, vm, vpci, basis, dev,
		util.ParisTime(c.TimeStamp.Time), p, bv,
		m.ShortMAs.P.Avg(), m.LongMAs.P.Avg(),
		m.ShortMAs.V.Avg(), m.LongMAs.V.Avg(),
		m.ShortMAs.PV.Avg(), m.LongMAs.PV.Avg(),
	)

	return hit
}

func (m *Market) StartPolling() {
	var (
		shortPoll = time.Duration(Candles[m.Interval]) / 6
		longPoll  = time.Duration(Candles[m.Interval]) * 4 / 5
		name      = m.MarketName
		timer     = time.NewTimer(shortPoll)
		c         bittrex.Candle
	)

	log.Println(shortPoll, longPoll, time.Duration(Candles[m.Interval]))
	m.stop = make(chan interface{})
	for {
		select {
		case <-m.stop:
			return
		case <-timer.C:
			// default to short poll
			timer.Reset(shortPoll)
		}

		candles, err := m.Client.GetLatestTick(name, string(m.Interval))
		if err != nil {
			log.Printf("bittrex GetLatestTick %s: %s", name, err)
			continue
		}
		c = candles[0]
		if !m.IsCandleNew(c) {
			log.Println("no new candle for", m)
			continue
		}

		// we have new value, long poll
		timer.Reset(longPoll)

		if m.AddCandle(c, false) {
			log.Printf("%18s HIT - consecutive: %d, total: %3d",
				m.MarketName, m.ConsecutiveHits, m.TotalHits)
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
