package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"sync"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

var priceclasses = []string{"SE1", "SE2", "SE3", "SE4"}

var influxaddr = flag.String("influxaddr", "http://localhost:8086", "InfluxDB address")
var influxtoken = flag.String("influxtoken", "my-token", "InfluxDB token")
var influxInterval = flag.Duration("influxupdaterate", time.Second*10, "InfluxDB datapoint injection rate")
var influxOrg = flag.String("influxorg", "my-org", "InfluxDB Organisation")
var influxBucket = flag.String("influxbucket", "my-bucket", "InfluxDB bucket")
var priceclass = flag.String("priceclass", "SE3", fmt.Sprintf("Priceclass, one of: %v", priceclasses))

var clocksource = time.Now

const (
	Locale  = "Europe/Stockholm"
	BaseURL = "https://www.elprisetjustnu.se"
)

type Price struct {
	SEKPerkWh float64   `json:"SEK_per_kWh"`
	EURPerkWh float64   `json:"EUR_per_kWh"`
	EXR       float64   `json:"EXR"`
	TimeStart time.Time `json:"time_start"`
	TimeEnd   time.Time `json:"time_end"`
}

type Prices []Price

func NewPriceClient(priceclass string) *PriceClient {
	return &PriceClient{
		baseURL:    BaseURL,
		priceclass: priceclass,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

type PriceClient struct {
	baseURL    string
	priceclass string
	client     *http.Client
	prices     Prices
	mutex      sync.Mutex
}

func (p *PriceClient) apiURL() string {
	loc, _ := time.LoadLocation(Locale)
	now := p.Now()
	return fmt.Sprintf("%s/api/v1/prices/%d/%s_%s.json",
		p.baseURL,
		now.In(loc).Year(),
		now.In(loc).Format("01-02"),
		p.priceclass,
	)
}

func (p *PriceClient) Now() time.Time {
	return clocksource()
}

// CurrentPriceSEK returns the price in SEK at this given time.
func (p *PriceClient) CurrentPriceSEK() (float64, error) {
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

// LoadPrices loads the prices for the active day into memory.
func (p *PriceClient) LoadPrices() error {
	resp, err := p.client.Get(p.apiURL())
	if err != nil {
		return fmt.Errorf("error reading from %s: %v", BaseURL, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
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
	fmt.Println("Prices loaded from", BaseURL)
	return nil
}

// PriceLoader handles the refresh of prices from the API at midnight.
func (p *PriceClient) PriceLoader() {
	loc, err := time.LoadLocation(Locale)
	if err != nil {
		log.Fatal(err)
	}
	for {
		now := p.Now()
		year, month, day := now.In(loc).Date()
		t := time.Date(year, month, day, 0, 0, 0, 0, loc).AddDate(0, 0, 1).Add(time.Second)
		fmt.Println("Time for price refresh:", t)
		<-time.After(t.Sub(p.Now()))
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
	if !slices.Contains(priceclasses, *priceclass) {
		log.Fatalf("Priceclass must be one of %v", priceclasses)
	}

	wg := sync.WaitGroup{}
	priceClient := NewPriceClient(*priceclass)

	// Load the prices once
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
	writeAPI := client.WriteAPIBlocking(*influxOrg, *influxBucket)

	ticker := time.NewTicker(*influxInterval)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			<-ticker.C
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), *influxInterval-(time.Millisecond*500))
				defer cancel()

				price, err := priceClient.CurrentPriceSEK()
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

	fmt.Println("Pushing prices to InfluxDB at update rate:", *influxInterval)
	wg.Wait()
}
