package main

import (
	"fmt"
	"github.com/montanaflynn/stats"
	"github.com/rkjdid/util"
	"github.com/shopspring/decimal"
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

	if sz > (len(candles) - 1) {
		sz = len(candles) - 1
	}

	for i := len(candles) - sz - 1; i < len(candles)-1; i++ {
		m.AddCandle(candles[i], true)
	}
	return nil
}

// AddCandle inserts c as the last candle, recomputing Values()
// and returning true if added candle was a hit.
func (m *Market) AddCandle(c bittrex.Candle, fillOnly bool) bool {
	m.LastCandle = c
	p := c.Close
	v := c.Volume
	m.ShortMAs.Add(p, v)
	m.LongMAs.Add(p, v)

	vpc := (m.LongMAs.PV.Avg().Div(m.LongMAs.V.Avg())).Sub(m.LongMAs.P.Avg())
	vpr := (m.ShortMAs.PV.Avg().Div(m.ShortMAs.V.Avg())).Div(m.ShortMAs.P.Avg())
	vm := m.ShortMAs.V.Avg().Div(m.LongMAs.V.Avg())
	vpci := vpc.Mul(vpr).Mul(vm)

	m.BBSum.Add(vpci)
	if fillOnly {
		return false
	}

	dev, err := stats.StandardDeviation(stats.Float64Data(m.BBSum.FloatValues()))
	if err != nil {
		log.Panicf("stdev shouldnt error: %s (len: %d)", err, len(m.BBSum.Values()))
	}

	basis := m.BBSum.Avg()
	dev *= m.Multiplier

	// hit detection
	hit := vpci.GreaterThan(basis.Add(decimal.NewFromFloat(dev)))

	if hit {
		m.ConsecutiveHits += 1
		m.TotalHits += 1
	} else {
		m.ConsecutiveHits = 0
	}

	log.Printf("%12s - %s - price %s - volume: %s - MA_P: %s / %s, MA_V: %s / %s, MA_PV: %s / %s\n\t"+
		"vpc: %s, vpr: %s, vm: %s, vpci: %s, basis: %s, dev: %f",
		m.MarketName, util.ParisTime(c.TimeStamp.Time), p, v,
		m.ShortMAs.P.Avg(), m.LongMAs.P.Avg(),
		m.ShortMAs.V.Avg(), m.LongMAs.V.Avg(),
		m.ShortMAs.PV.Avg(), m.LongMAs.PV.Avg(),
		vpc, vpr, vm, vpci, basis, dev)

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

	m.stop = make(chan interface{})
	for {
		candles, err := m.Client.GetLatestTick(name, string(m.Interval))
		if err != nil {
			log.Printf("bittrex GetLatestTick %s: %s", name, err)
			timer.Reset(shortPoll)
			goto sleep
		}
		c = candles[0]
		if !m.IsCandleNew(c) {
			timer.Reset(shortPoll)
			goto sleep
		}

		if m.AddCandle(c, false) {
			log.Printf("%18s HIT - consecutive: %d, total: %3d",
				m.MarketName, m.ConsecutiveHits, m.TotalHits)
		}
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
