package main

import (
	"github.com/montanaflynn/stats"
	"github.com/rkjdid/bitbot/movingaverage"
	"github.com/toorop/go-bittrex"
	"log"
	"strings"
	"sync"
	"time"
)

type Scanner struct {
	Config *ScannerConfig

	Markets map[string]*Market
	client  *bittrex.Bittrex
	stop    chan interface{}
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

		if s.Config.Pairs == nil || len(s.Config.Pairs) == 0 {
			m := &Market{
				Market:      market,
				MA_V_Long:   movingaverage.New(s.Config.LongTerm),
				MA_P_Long:   movingaverage.New(s.Config.LongTerm),
				MA_PV_Long:  movingaverage.New(s.Config.LongTerm),
				MA_V_Short:  movingaverage.New(s.Config.ShortTerm),
				MA_P_Short:  movingaverage.New(s.Config.ShortTerm),
				MA_PV_Short: movingaverage.New(s.Config.ShortTerm),

				BBSum: movingaverage.New(s.Config.BBLength),
			}
			s.Markets[market.MarketName] = m
		}

		for _, pair := range s.Config.Pairs {
			if pair == market.MarketName {
				m := &Market{
					Market:      market,
					MA_V_Long:   movingaverage.New(s.Config.LongTerm),
					MA_P_Long:   movingaverage.New(s.Config.LongTerm),
					MA_PV_Long:  movingaverage.New(s.Config.LongTerm),
					MA_V_Short:  movingaverage.New(s.Config.ShortTerm),
					MA_P_Short:  movingaverage.New(s.Config.ShortTerm),
					MA_PV_Short: movingaverage.New(s.Config.ShortTerm),

					BBSum: movingaverage.New(s.Config.BBLength),
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
	ticker := time.NewTicker(time.Duration(Candles[s.Config.Candle]))
	s.client = bittrex.New("", "")
	s.Markets = make(map[string]*Market)
	s.fetchMarkets()
	s.stop = make(chan interface{}, 1)

	for {
		for name, market := range s.Markets {
			go func(name string, market *Market) {
				candles, err := s.client.GetLatestTick(name, s.Config.Candle)
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
				dev *= s.Config.Multiplier

				if vpci > (basis + dev) {
					market.ConsecutiveHits += 1
					market.TotalHits += 1
					log.Printf("--------- %18s HIT - price: %8f, consecutive: %d, total: %3d",
						market.MarketName, p, market.ConsecutiveHits, market.TotalHits)
				} else {
					market.ConsecutiveHits = 0
				}
				log.Printf("%15s - vpc: %8f, vpr: %8f, vm: %8f, vpci: %8f, basis: %8f, dev: %8f\n\t\t"+
					"price: %8f, base_volume: %8f, MA_P: %8f / %8f, MA_V: %8f / %8f, MA_PV: %8f / %8f\n\t\tcandle: %#v\n\n",
					market.MarketName, vpc, vpc, vm, vpci, basis, dev,
					p, v,
					market.MA_P_Short.Avg(), market.MA_P_Long.Avg(),
					market.MA_V_Short.Avg(), market.MA_V_Long.Avg(),
					market.MA_PV_Short.Avg(), market.MA_PV_Long.Avg(),
					candle,
				)
			}(name, market)
		}

		select {
		case <-ticker.C:
		case <-s.stop:
			return
		}
	}
}
