package main

import (
	"github.com/montanaflynn/stats"
	"github.com/rkjdid/util"
	"github.com/toorop/go-bittrex"
	"log"
	"strings"
	"time"
)

type Scanner struct {
	Config *ScannerConfig

	Markets map[string]*Market
	client  *bittrex.Bittrex
	stop    chan interface{}
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

		trackMarket := func(name string) {
			s.Markets[name] =
				NewMarket(market, s.Config.LongTerm, s.Config.ShortTerm, s.Config.BBLength, s.Config.Interval)
		}

		name := market.MarketName
		if s.Config.Pairs == nil || len(s.Config.Pairs) == 0 {
			// no filter, track all markets
			trackMarket(name)
		} else {
			// else only track this market if it is in Config.Pairs filter
			for _, pair := range s.Config.Pairs {
				if pair == name {
					trackMarket(name)
					break
				}
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
	ticker := time.NewTicker(time.Duration(Candles[s.Config.Interval]))
	s.client = bittrex.New("", "")
	s.Markets = make(map[string]*Market)
	s.fetchMarkets()
	s.stop = make(chan interface{}, 1)

	for {
		for name, market := range s.Markets {
			// short throttle to avoid spawning to many goroutines at once
			go func(name string, market *Market) {
				var candle = market.LastCandle
				for {
					time.After(time.Second)
					candles, err := s.client.GetLatestTick(name, string(market.Interval))
					if err != nil {
						log.Printf("bittrex GetLatestTick %s: %s", name, err)
						return
					}
					candle = candles[0]
					if candle.TimeStamp.Time != market.LastCandle.TimeStamp.Time {
						break
					}
					<-time.After(time.Second * 15)
				}

				market.LastCandle = candle
				p, _ := candle.Close.Float64()
				v, _ := candle.Volume.Float64()
				market.ShortMAs.Add(p, v)
				market.LongMAs.Add(p, v)

				vpc := market.LongMAs.PV.Avg()/market.LongMAs.V.Avg() - market.LongMAs.P.Avg()
				vpr := market.ShortMAs.PV.Avg() / market.ShortMAs.V.Avg() / market.ShortMAs.P.Avg()
				vm := market.ShortMAs.V.Avg() / market.LongMAs.V.Avg()
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

				bv, _ := candle.BaseVolume.Float64()
				log.Printf("%15s - vpc: %8f, vpr: %8f, vm: %8f, vpci: %8f, basis: %8f, dev: %8f\n\t\t"+
					"candle: %s, price: %8f, btc_vol: %8f, MA_P: %8f / %8f, MA_V: %8f / %8f, MA_PV: %8f / %8f",
					market.MarketName, vpc, vpr, vm, vpci, basis, dev,
					util.ParisTime(candle.TimeStamp.Time), p, bv,
					market.ShortMAs.P.Avg(), market.LongMAs.P.Avg(),
					market.ShortMAs.V.Avg(), market.LongMAs.V.Avg(),
					market.ShortMAs.PV.Avg(), market.LongMAs.PV.Avg(),
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
