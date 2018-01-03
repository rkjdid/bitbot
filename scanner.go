package main

import (
	"github.com/montanaflynn/stats"
	"github.com/rkjdid/bitbot/movingaverage"
	"github.com/rkjdid/util"
	"github.com/toorop/go-bittrex"
	"log"
	"strings"
	"time"
)

var Candles = map[string]util.Duration{
	"hour": util.Duration(time.Hour),
}

const CandleHour = "hour"

type Scanner struct {
	Pairs   []string
	Candle  string
	Markets map[string]*Market

	LongTerm   int
	ShortTerm  int
	BBLength   int
	Multiplier float64

	NotificationThreshold int // number of consecutive hits to trigger notification

	client *bittrex.Bittrex
	stop   chan interface{}
}

type MATrio struct {
	Length      int
	Price       *movingaverage.MovingAverage
	Volume      *movingaverage.MovingAverage
	PriceVolume *movingaverage.MovingAverage
}

type Market struct {
	bittrex.Market
	Candles []bittrex.Candle

	MA_P_Long   *movingaverage.MovingAverage
	MA_P_Short  *movingaverage.MovingAverage
	MA_PV_Long  *movingaverage.MovingAverage
	MA_PV_Short *movingaverage.MovingAverage
	MA_V_Long   *movingaverage.MovingAverage
	MA_V_Short  *movingaverage.MovingAverage

	BBSum *movingaverage.MovingAverage

	ConsecutiveHits int
	TotalHits       int
}

func (s *Scanner) fetchMarkets() error {
	markets, err := s.client.GetMarkets()
	if err != nil {
		return err
	}

	for _, market := range markets {
		if !market.IsActive {
			// only monitor active markets
			continue
		}
		if strings.Index(market.MarketName, "BTC") == -1 {
			// only monitor btc markets
			continue
		}

		for _, pair := range s.Pairs {
			if pair == market.MarketName {
				m := &Market{
					Market:      market,
					MA_V_Long:   movingaverage.New(s.LongTerm),
					MA_P_Long:   movingaverage.New(s.LongTerm),
					MA_PV_Long:  movingaverage.New(s.LongTerm),
					MA_V_Short:  movingaverage.New(s.ShortTerm),
					MA_P_Short:  movingaverage.New(s.ShortTerm),
					MA_PV_Short: movingaverage.New(s.ShortTerm),

					BBSum: movingaverage.New(s.BBLength),
				}
				s.Markets[market.MarketName] = m
			}
		}
	}
	return nil
}

func (s *Scanner) Stop() {
	if s.stop == nil {
		return
	}
	s.stop <- nil
	s.stop = nil
}

func (s *Scanner) Scan() {
	ticker := time.NewTicker(time.Duration(Candles[s.Candle]))
	s.client = bittrex.New("", "")
	s.Markets = make(map[string]*Market)
	s.fetchMarkets()
	s.stop = make(chan interface{}, 1)

	for {
		for name, market := range s.Markets {
			go func(name string, market *Market) {
				candles, err := s.client.GetLatestTick(name, s.Candle)
				if err != nil {
					log.Printf("bittrex GetLatestTick %s: %s", name, err)
					return
				}

				candle := candles[0]
				p, _ := candle.Close.Float64()
				v, _ := candle.Volume.Float64()
				market.MA_P_Long.Add(p)
				market.MA_P_Short.Add(p)
				market.MA_PV_Long.Add(p * v)
				market.MA_PV_Short.Add(p * v)
				market.MA_V_Long.Add(v)
				market.MA_V_Short.Add(v)

				vpc := market.MA_PV_Long.Avg()/market.MA_V_Long.Avg() - market.MA_P_Long.Avg()
				vpr := (market.MA_PV_Short.Avg() / market.MA_V_Short.Avg()) / market.MA_P_Short.Avg()
				vm := market.MA_V_Short.Avg() / market.MA_V_Long.Avg()
				vpci := vpc * vpr * vm

				market.BBSum.Add(vpci)
				dev, err := stats.StandardDeviation(stats.Float64Data(market.BBSum.Values))
				if err != nil {
					log.Printf("%s - stats.StandardDeviation error: %s", market.MarketName, err)
					return
				}

				basis := market.BBSum.Avg()
				dev *= s.Multiplier
				if vpci > (basis + dev) {
					market.ConsecutiveHits += 1
					market.TotalHits += 1
					log.Printf("%8s hit - basis: %8.2f, dev: %8.2f, consecutive: %d, total: %3d",
						market.MarketName, basis, dev, market.ConsecutiveHits, market.TotalHits)
				} else {
					market.ConsecutiveHits = 0
				}
				log.Printf("%8s - vpc: %5.1f, vpr: %5.1f, vm: %5.1f, basis: %5.1f, dev: %5.1f",
					market.MarketName, vpc, vpc, vm, basis, dev)
			}(name, market)
		}

		select {
		case <-ticker.C:
		case <-s.stop:
			return
		}
	}
}
