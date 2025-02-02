// Bwulfs super-price-pusher-3000
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

var influxaddr = flag.String("influxaddr", "http://localhost:8086", "InfluxDB address")
var influxtoken = flag.String("influxtoken", "my-token", "InfluxDB token")
var influxInterval = flag.Duration("influxupdaterate", time.Second*10, "InfluxDB datapoint injection rate, defaults to 10s")

const (
	InfluxOrg    = "my-org"
	InfluxBucket = "my-bucket"
	Locale       = "Europe/Stockholm"
	BaseURL      = "https://www.elprisetjustnu.se"
)

// {"SEK_per_kWh":0.4931,"EUR_per_kWh":0.04295,"EXR":11.480681,"time_start":"2025-01-29T00:00:00+01:00","time_end":"2025-01-29T01:00:00+01:00"}
type Price struct {
	SEKPerkWh float64   `json:"SEK_per_kWh"`
	EURPerkWh float64   `json:"EUR_per_kWh"`
	EXR       float64   `json:"EXR"`
	TimeStart time.Time `json:"time_start"`
	TimeEnd   time.Time `json:"time_end"`
}

type Prices []Price

func NewPriceClient() *PriceClient {
	return &PriceClient{
		baseURL:     BaseURL,
		client:      &http.Client{Timeout: 10 * time.Second},
		clocksource: time.Now,
	}
}

type PriceClient struct {
	baseURL     string
	client      *http.Client
	prices      Prices
	mutex       sync.Mutex
	clocksource func() time.Time
}

func (p *PriceClient) createElprisURL() string {
	loc, _ := time.LoadLocation(Locale)
	now := p.Now()
	return fmt.Sprintf("%s/api/v1/prices/%d/%s_SE3.json",
		p.baseURL,
		now.In(loc).Year(),
		now.In(loc).Format("01-02"),
	)
}

func (p *PriceClient) Now() time.Time {
	return p.clocksource()
}

func (p *PriceClient) GetCurrentPrice() (float64, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	now := p.Now()
	for _, price := range p.prices {
		if price.TimeStart.Before(now) && price.TimeEnd.After(now) {
			return price.SEKPerkWh, nil
		}
	}
	return 0, fmt.Errorf("no current price found, no fresh data?")
}

func (p *PriceClient) LoadPrices() error {
	resp, err := p.client.Get(p.createElprisURL())
	if err != nil {
		return fmt.Errorf("error reading from elprisetjustnu.se: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body) // response body is []byte
	if err != nil {
		return fmt.Errorf("error reading response: %v", err)
	}
	var prices Prices
	err = json.Unmarshal(body, &prices)
	if err != nil {
		return fmt.Errorf("error parsing json: %v", err)
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.prices = prices
	fmt.Println("Prices loaded from elprisetjustnu.se")
	return nil
}

func (p *PriceClient) PriceLoader() {
	for {
		loc, err := time.LoadLocation(Locale)
		if err != nil {
			log.Fatal(err)
		}
		now := p.clocksource()
		year, month, day := now.In(loc).Date()
		t := time.Date(year, month, day, 0, 0, 0, 0, loc).AddDate(0, 0, 1).Add(time.Second)
		fmt.Println("New prices will be fetched at:", t)
		<-time.After(t.Sub(p.clocksource()))
		fmt.Println("Fetching new prices from the API")
		err = p.LoadPrices()
		if err != nil {
			fmt.Println("Error loading prices", err)
			continue
		}
	}
}

func main() {
	flag.Parse()
	wg := sync.WaitGroup{}
	priceClient := NewPriceClient()
	err := priceClient.LoadPrices()
	if err != nil {
		log.Fatal(err)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		priceClient.PriceLoader()
	}()

	client := influxdb2.NewClient(*influxaddr, *influxtoken)
	writeAPI := client.WriteAPIBlocking(InfluxOrg, InfluxBucket)

	ticker := time.NewTicker(*influxInterval)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			<-ticker.C
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), *influxInterval-(time.Millisecond*500))
				defer cancel()
				// Create point using fluent style
				price, err := priceClient.GetCurrentPrice()
				if err != nil {
					log.Printf("GetCurrentPrice: %v", err)
					return
				}
				p := influxdb2.NewPointWithMeasurement("price").
					AddTag("currency", "SEK").
					AddField("price", price).
					SetTime(time.Now())
				err = writeAPI.WritePoint(ctx, p)
				if err != nil {
					log.Printf("Write to influx failed: %v", err)
				}
			}()
		}
	}()

	fmt.Println("Started with InfluxDB update rate:", *influxInterval)
	wg.Wait()
}
