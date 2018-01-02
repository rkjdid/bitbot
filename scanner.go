package main

import (
	"github.com/rkjdid/util"
	"github.com/shopspring/decimal"
	"github.com/toorop/go-bittrex"
	"log"
	"strings"
	"time"
)

type Scanner struct {
	Pairs          []string
	Ticker         util.Duration
	AssessmentsLen int
	Markets        map[string]*Market
	client         *bittrex.Bittrex
	stop           chan interface{}
}

type Market struct {
	bittrex.Market
	LastSnapshot bittrex.MarketSummary
	Assessments  []Assessment
	GlobalScore  decimal.Decimal
}

func (m *Market) Assess() {
	m.GlobalScore = decimal.Decimal{}
	for _, a := range m.Assessments {
		m.GlobalScore = m.GlobalScore.Add(a.Score)
	}
}

type Assessment struct {
	Snapshot_t0 bittrex.MarketSummary
	Snapshot_tN bittrex.MarketSummary
	Volume      decimal.Decimal // volume progression between t0 & tN
	Price       decimal.Decimal // price progression between t0 & tN
	Score       decimal.Decimal
}

func Assess(t0 bittrex.MarketSummary, t bittrex.MarketSummary) Assessment {
	//log.Printf("assessing %s: \n\t%v\n\t%v", t0.MarketName, t0, t)

	if t0.Volume.Equals(decimal.New(0, 0)) {
		return Assessment{}
	}

	a := Assessment{
		Snapshot_t0: t0,
		Snapshot_tN: t,
		Volume:      t.BaseVolume.Div(t0.BaseVolume).Sub(decimal.New(1, 0)).Mul(decimal.New(100, 0)),
		Price:       t.Last.Div(t0.Last).Sub(decimal.New(1, 0)).Mul(decimal.New(100, 0)),
	}

	a.Score = a.Volume.Add(a.Price)
	return a
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
				m := &Market{Market: market}
				m.Assessments = []Assessment{}
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
	ticker := time.NewTicker(time.Duration(s.Ticker))
	s.client = bittrex.New("", "")
	s.Markets = make(map[string]*Market)
	s.fetchMarkets()
	s.stop = make(chan interface{}, 1)

	for {
		for name, market := range s.Markets {
			go func(name string, market *Market) {
				summaries, err := s.client.GetMarketSummary(name)
				if err != nil {
					log.Printf("bittrex GetMarketSummary %s: %s", name, err)
					return
				}

				tN := summaries[0]
				t0 := market.LastSnapshot
				market.LastSnapshot = tN
				a := Assess(t0, tN)
				if len(market.Assessments) == s.AssessmentsLen {
					// slice full, pop first elem
					market.Assessments = market.Assessments[1:]
				}
				market.Assessments = append(market.Assessments, a)
				market.Assess()
				log.Printf("%s - global: %s, price: %s%%, volume: %s%%, score: %s, len: %d",
					name, market.GlobalScore.StringFixed(1), a.Price.StringFixed(1),
					a.Volume.StringFixed(1), a.Score.StringFixed(1), len(market.Assessments))
			}(name, market)
		}

		select {
		case <-ticker.C:
		case <-s.stop:
			return
		}
		log.Println("--------------")
	}
}
