package main

import (
	"github.com/toorop/go-bittrex"
	"log"
	"strings"
)

type Scanner struct {
	Config  *ScannerConfig
	Markets map[string]*Market
	Client  *bittrex.Bittrex
	stop    chan interface{}
}

func (s *Scanner) NewMarket(market bittrex.Market) *Market {
	return NewMarket(market, s.Config.ShortTerm, s.Config.LongTerm,
		s.Config.BBLength, s.Config.Interval, s.Config.Multiplier, s.Client)
}

func (s *Scanner) fetchMarkets() error {
	markets, err := s.Client.GetMarkets()
	if err != nil {
		return err
	}

	for _, market := range markets {
		name := market.MarketName
		if !market.IsActive {
			// only monitor active markets
			continue
		}
		if strings.Index(name, "BTC") != 0 {
			// only monitor btc markets
			continue
		}

		summary, err := s.Client.GetMarketSummary(name)
		if err != nil {
			log.Printf("error retreiving market history for %s: %s", market.MarketName, err)
			continue
		}

		bv, _ := summary[0].BaseVolume.Float64()
		if s.Config.MinBtcVolumeDaily < bv {
			// filter out low volume markets
			log.Printf("filtering out low volume market %s", market)
			continue
		}

		initMarket := func(market bittrex.Market) {
			m := s.NewMarket(market)
			err := m.FillCandles()
			if err != nil {
				log.Printf("error filling candles for %s: %s", market.MarketName, err)
				return
			}
			s.Markets[market.MarketName] = m
		}

		if s.Config.Pairs == nil || len(s.Config.Pairs) == 0 {
			// no filter, track all markets
			initMarket(market)
		} else {
			// else only track this market if it is in Config.Pairs filter
			for _, pair := range s.Config.Pairs {
				if pair == name {
					initMarket(market)
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
	s.Client = bittrex.New("", "")
	s.Markets = make(map[string]*Market)
	s.fetchMarkets()

	for _, market := range s.Markets {
		go market.StartPolling()
	}

	s.stop = make(chan interface{}, 1)
	select {
	case <-s.stop:
		for _, market := range s.Markets {
			market.Stop()
		}
		return
	}
}
