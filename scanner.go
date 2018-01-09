package main

import (
	"github.com/toorop/go-bittrex"
	"log"
	"strings"
	"time"
)

type Scanner struct {
	Config  *ScannerConfig
	Markets map[string]*Market
	Client  *bittrex.Bittrex
	stop    chan interface{}
}

func NewScanner(cfg *ScannerConfig) *Scanner {
	return &Scanner{
		Config:  cfg,
		Client:  bittrex.New("", ""),
		Markets: make(map[string]*Market),
	}
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

		if s.Config.Pairs != nil && len(s.Config.Pairs) > 0 {
			// only track this market if it is in Config.Pairs filter
			trackMarket := false
			for _, pair := range s.Config.Pairs {
				if pair == market.MarketName {
					trackMarket = true
					break
				}
			}
			if !trackMarket {
				continue
			}
		} else {
			if strings.Index(name, "BTC") != 0 {
				// only monitor btc markets
				continue
			}

			summary, err := s.Client.GetMarketSummary(name)
			if err != nil {
				log.Printf("error retreiving market history for %s: %s", name, err)
				continue
			}

			bv, _ := summary[0].BaseVolume.Float64()
			if bv < s.Config.MinBtcVolumeDaily {
				// filter out low volume markets
				log.Printf("filtering out low volume market %s (base vol: %5f)", name, bv)
				continue
			}
		}

		m := s.NewMarket(market)
		err = m.FillCandles()
		if err != nil {
			log.Printf("error filling candles for %s: %s", market.MarketName, err)
			continue
		}
		s.Markets[name] = m
		log.Println("tracking market", name)
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
	s.fetchMarkets()
	log.Printf("%d tracked markets", len(s.Markets))
	for _, market := range s.Markets {
		go market.StartPolling()
	}

	s.stop = make(chan interface{})
	select {
	case <-s.stop:
		for _, market := range s.Markets {
			market.Stop()
		}
		return
	}
}

func (s *Scanner) Analyze(marketName string, from, to time.Time) error {
	log.Printf("starting market analysis for \"%s\"", marketName)
	var bFrom, bTo bool
	if !from.Equal(time.Time{}) {
		bFrom = true
		log.Printf("from: %s", from)
	}
	if !to.Equal(time.Time{}) {
		bTo = true
		log.Printf("  to: %s", to)
	}

	m := s.NewMarket(bittrex.Market{MarketName: marketName})
	candles, err := s.Client.GetTicks(marketName, string(s.Config.Interval))
	if err != nil {
		return err
	}

	for _, c := range candles {
		// candles should come in historical order..
		if bFrom && c.TimeStamp.Time.Before(from) {
			continue
		}
		if bTo && c.TimeStamp.Time.After(to) {
			break
		}

		_ = m.AddCandle(c, false)
	}

	return nil
}
