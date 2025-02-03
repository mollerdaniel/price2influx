package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

type fakeclock struct {
	curtime time.Time
}

func (f *fakeclock) Now() time.Time {
	return f.curtime
}

const (
	day1 = `[{"SEK_per_kWh":0.37003,"EUR_per_kWh":0.03218,"EXR":11.498678,"time_start":"2025-02-02T00:00:00+01:00","time_end":"2025-02-02T01:00:00+01:00"},{"SEK_per_kWh":0.34508,"EUR_per_kWh":0.03001,"EXR":11.498678,"time_start":"2025-02-02T01:00:00+01:00","time_end":"2025-02-02T02:00:00+01:00"},{"SEK_per_kWh":0.34439,"EUR_per_kWh":0.02995,"EXR":11.498678,"time_start":"2025-02-02T02:00:00+01:00","time_end":"2025-02-02T03:00:00+01:00"},{"SEK_per_kWh":0.34312,"EUR_per_kWh":0.02984,"EXR":11.498678,"time_start":"2025-02-02T03:00:00+01:00","time_end":"2025-02-02T04:00:00+01:00"},{"SEK_per_kWh":0.34059,"EUR_per_kWh":0.02962,"EXR":11.498678,"time_start":"2025-02-02T04:00:00+01:00","time_end":"2025-02-02T05:00:00+01:00"},{"SEK_per_kWh":0.36037,"EUR_per_kWh":0.03134,"EXR":11.498678,"time_start":"2025-02-02T05:00:00+01:00","time_end":"2025-02-02T06:00:00+01:00"},{"SEK_per_kWh":0.47191,"EUR_per_kWh":0.04104,"EXR":11.498678,"time_start":"2025-02-02T06:00:00+01:00","time_end":"2025-02-02T07:00:00+01:00"},{"SEK_per_kWh":0.77697,"EUR_per_kWh":0.06757,"EXR":11.498678,"time_start":"2025-02-02T07:00:00+01:00","time_end":"2025-02-02T08:00:00+01:00"},{"SEK_per_kWh":0.81629,"EUR_per_kWh":0.07099,"EXR":11.498678,"time_start":"2025-02-02T08:00:00+01:00","time_end":"2025-02-02T09:00:00+01:00"},{"SEK_per_kWh":0.82296,"EUR_per_kWh":0.07157,"EXR":11.498678,"time_start":"2025-02-02T09:00:00+01:00","time_end":"2025-02-02T10:00:00+01:00"},{"SEK_per_kWh":0.78352,"EUR_per_kWh":0.06814,"EXR":11.498678,"time_start":"2025-02-02T10:00:00+01:00","time_end":"2025-02-02T11:00:00+01:00"},{"SEK_per_kWh":0.78628,"EUR_per_kWh":0.06838,"EXR":11.498678,"time_start":"2025-02-02T11:00:00+01:00","time_end":"2025-02-02T12:00:00+01:00"},{"SEK_per_kWh":0.74902,"EUR_per_kWh":0.06514,"EXR":11.498678,"time_start":"2025-02-02T12:00:00+01:00","time_end":"2025-02-02T13:00:00+01:00"},{"SEK_per_kWh":0.69015,"EUR_per_kWh":0.06002,"EXR":11.498678,"time_start":"2025-02-02T13:00:00+01:00","time_end":"2025-02-02T14:00:00+01:00"},{"SEK_per_kWh":0.70958,"EUR_per_kWh":0.06171,"EXR":11.498678,"time_start":"2025-02-02T14:00:00+01:00","time_end":"2025-02-02T15:00:00+01:00"},{"SEK_per_kWh":0.77892,"EUR_per_kWh":0.06774,"EXR":11.498678,"time_start":"2025-02-02T15:00:00+01:00","time_end":"2025-02-02T16:00:00+01:00"},{"SEK_per_kWh":0.91736,"EUR_per_kWh":0.07978,"EXR":11.498678,"time_start":"2025-02-02T16:00:00+01:00","time_end":"2025-02-02T17:00:00+01:00"},{"SEK_per_kWh":1.01292,"EUR_per_kWh":0.08809,"EXR":11.498678,"time_start":"2025-02-02T17:00:00+01:00","time_end":"2025-02-02T18:00:00+01:00"},{"SEK_per_kWh":1.06351,"EUR_per_kWh":0.09249,"EXR":11.498678,"time_start":"2025-02-02T18:00:00+01:00","time_end":"2025-02-02T19:00:00+01:00"},{"SEK_per_kWh":1.14251,"EUR_per_kWh":0.09936,"EXR":11.498678,"time_start":"2025-02-02T19:00:00+01:00","time_end":"2025-02-02T20:00:00+01:00"},{"SEK_per_kWh":0.99809,"EUR_per_kWh":0.0868,"EXR":11.498678,"time_start":"2025-02-02T20:00:00+01:00","time_end":"2025-02-02T21:00:00+01:00"},{"SEK_per_kWh":0.92553,"EUR_per_kWh":0.08049,"EXR":11.498678,"time_start":"2025-02-02T21:00:00+01:00","time_end":"2025-02-02T22:00:00+01:00"},{"SEK_per_kWh":0.59885,"EUR_per_kWh":0.05208,"EXR":11.498678,"time_start":"2025-02-02T22:00:00+01:00","time_end":"2025-02-02T23:00:00+01:00"},{"SEK_per_kWh":0.4634,"EUR_per_kWh":0.0403,"EXR":11.498678,"time_start":"2025-02-02T23:00:00+01:00","time_end":"2025-02-03T00:00:00+01:00"}]`
	day2 = `[{"SEK_per_kWh":0.48455,"EUR_per_kWh":0.04214,"EXR":11.498678,"time_start":"2025-02-03T00:00:00+01:00","time_end":"2025-02-03T01:00:00+01:00"},{"SEK_per_kWh":0.40774,"EUR_per_kWh":0.03546,"EXR":11.498678,"time_start":"2025-02-03T01:00:00+01:00","time_end":"2025-02-03T02:00:00+01:00"},{"SEK_per_kWh":0.40809,"EUR_per_kWh":0.03549,"EXR":11.498678,"time_start":"2025-02-03T02:00:00+01:00","time_end":"2025-02-03T03:00:00+01:00"},{"SEK_per_kWh":0.40418,"EUR_per_kWh":0.03515,"EXR":11.498678,"time_start":"2025-02-03T03:00:00+01:00","time_end":"2025-02-03T04:00:00+01:00"},{"SEK_per_kWh":0.43545,"EUR_per_kWh":0.03787,"EXR":11.498678,"time_start":"2025-02-03T04:00:00+01:00","time_end":"2025-02-03T05:00:00+01:00"},{"SEK_per_kWh":0.59287,"EUR_per_kWh":0.05156,"EXR":11.498678,"time_start":"2025-02-03T05:00:00+01:00","time_end":"2025-02-03T06:00:00+01:00"},{"SEK_per_kWh":1.17815,"EUR_per_kWh":0.10246,"EXR":11.498678,"time_start":"2025-02-03T06:00:00+01:00","time_end":"2025-02-03T07:00:00+01:00"},{"SEK_per_kWh":1.71169,"EUR_per_kWh":0.14886,"EXR":11.498678,"time_start":"2025-02-03T07:00:00+01:00","time_end":"2025-02-03T08:00:00+01:00"},{"SEK_per_kWh":1.88912,"EUR_per_kWh":0.16429,"EXR":11.498678,"time_start":"2025-02-03T08:00:00+01:00","time_end":"2025-02-03T09:00:00+01:00"},{"SEK_per_kWh":1.72066,"EUR_per_kWh":0.14964,"EXR":11.498678,"time_start":"2025-02-03T09:00:00+01:00","time_end":"2025-02-03T10:00:00+01:00"},{"SEK_per_kWh":1.72101,"EUR_per_kWh":0.14967,"EXR":11.498678,"time_start":"2025-02-03T10:00:00+01:00","time_end":"2025-02-03T11:00:00+01:00"},{"SEK_per_kWh":1.45136,"EUR_per_kWh":0.12622,"EXR":11.498678,"time_start":"2025-02-03T11:00:00+01:00","time_end":"2025-02-03T12:00:00+01:00"},{"SEK_per_kWh":1.34891,"EUR_per_kWh":0.11731,"EXR":11.498678,"time_start":"2025-02-03T12:00:00+01:00","time_end":"2025-02-03T13:00:00+01:00"},{"SEK_per_kWh":1.32499,"EUR_per_kWh":0.11523,"EXR":11.498678,"time_start":"2025-02-03T13:00:00+01:00","time_end":"2025-02-03T14:00:00+01:00"},{"SEK_per_kWh":1.4294,"EUR_per_kWh":0.12431,"EXR":11.498678,"time_start":"2025-02-03T14:00:00+01:00","time_end":"2025-02-03T15:00:00+01:00"},{"SEK_per_kWh":1.67846,"EUR_per_kWh":0.14597,"EXR":11.498678,"time_start":"2025-02-03T15:00:00+01:00","time_end":"2025-02-03T16:00:00+01:00"},{"SEK_per_kWh":1.95225,"EUR_per_kWh":0.16978,"EXR":11.498678,"time_start":"2025-02-03T16:00:00+01:00","time_end":"2025-02-03T17:00:00+01:00"},{"SEK_per_kWh":2.47038,"EUR_per_kWh":0.21484,"EXR":11.498678,"time_start":"2025-02-03T17:00:00+01:00","time_end":"2025-02-03T18:00:00+01:00"},{"SEK_per_kWh":2.31319,"EUR_per_kWh":0.20117,"EXR":11.498678,"time_start":"2025-02-03T18:00:00+01:00","time_end":"2025-02-03T19:00:00+01:00"},{"SEK_per_kWh":2.30939,"EUR_per_kWh":0.20084,"EXR":11.498678,"time_start":"2025-02-03T19:00:00+01:00","time_end":"2025-02-03T20:00:00+01:00"},{"SEK_per_kWh":1.8882,"EUR_per_kWh":0.16421,"EXR":11.498678,"time_start":"2025-02-03T20:00:00+01:00","time_end":"2025-02-03T21:00:00+01:00"},{"SEK_per_kWh":1.49437,"EUR_per_kWh":0.12996,"EXR":11.498678,"time_start":"2025-02-03T21:00:00+01:00","time_end":"2025-02-03T22:00:00+01:00"},{"SEK_per_kWh":1.24347,"EUR_per_kWh":0.10814,"EXR":11.498678,"time_start":"2025-02-03T22:00:00+01:00","time_end":"2025-02-03T23:00:00+01:00"},{"SEK_per_kWh":0.61023,"EUR_per_kWh":0.05307,"EXR":11.498678,"time_start":"2025-02-03T23:00:00+01:00","time_end":"2025-02-04T00:00:00+01:00"}]`
)

func TestMidnightLoading(t *testing.T) {
	asdf, err := time.LoadLocation("America/Aruba")
	if err != nil {
		t.Fatal(err)
	}
	time.Local = asdf
	loc, err := time.LoadLocation(Locale)
	if err != nil {
		t.Fatal(err)
	}

	fakec := fakeclock{
		curtime: time.Date(2025, 2, 2, 23, 59, 59, 0, loc),
	}

	ts1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/prices/2025/02-02_SE3.json" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		fmt.Fprintln(w, day1)
	}))
	defer ts1.Close()
	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/prices/2025/02-03_SE3.json" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		fmt.Fprintln(w, day2)
	}))
	defer ts2.Close()

	clocksource = fakec.Now
	pc := PriceClient{
		baseURL:    ts1.URL,
		client:     ts1.Client(),
		priceclass: "SE3",
	}
	err = pc.LoadPrices()
	if err != nil {
		t.Errorf("LoadPrices() error = %v, wantErr nil", err)
		return
	}

	pc.baseURL = ts2.URL
	pc.client = ts2.Client()
	fakec.curtime = time.Date(2025, 2, 3, 0, 0, 0, 1, loc)

	err = pc.LoadPrices()
	if err != nil {
		t.Errorf("LoadPrices() error = %v, wantErr nil", err)
		return
	}
}

func TestLoadPrices(t *testing.T) {
	loc, err := time.LoadLocation(Locale)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		response   string
		wantErr    bool
		wantPrices Prices
	}{
		{
			name:     "test day 1",
			response: day1,
			wantErr:  false,
			wantPrices: Prices{
				{SEKPerkWh: 0.37003, EURPerkWh: 0.03218, EXR: 11.498678, TimeStart: time.Date(2025, 2, 2, 0, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 2, 1, 0, 0, 0, loc)},
				{SEKPerkWh: 0.34508, EURPerkWh: 0.03001, EXR: 11.498678, TimeStart: time.Date(2025, 2, 2, 1, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 2, 2, 0, 0, 0, loc)},
				{SEKPerkWh: 0.34439, EURPerkWh: 0.02995, EXR: 11.498678, TimeStart: time.Date(2025, 2, 2, 2, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 2, 3, 0, 0, 0, loc)},
				{SEKPerkWh: 0.34312, EURPerkWh: 0.02984, EXR: 11.498678, TimeStart: time.Date(2025, 2, 2, 3, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 2, 4, 0, 0, 0, loc)},
				{SEKPerkWh: 0.34059, EURPerkWh: 0.02962, EXR: 11.498678, TimeStart: time.Date(2025, 2, 2, 4, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 2, 5, 0, 0, 0, loc)},
				{SEKPerkWh: 0.36037, EURPerkWh: 0.03134, EXR: 11.498678, TimeStart: time.Date(2025, 2, 2, 5, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 2, 6, 0, 0, 0, loc)},
				{SEKPerkWh: 0.47191, EURPerkWh: 0.04104, EXR: 11.498678, TimeStart: time.Date(2025, 2, 2, 6, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 2, 7, 0, 0, 0, loc)},
				{SEKPerkWh: 0.77697, EURPerkWh: 0.06757, EXR: 11.498678, TimeStart: time.Date(2025, 2, 2, 7, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 2, 8, 0, 0, 0, loc)},
				{SEKPerkWh: 0.81629, EURPerkWh: 0.07099, EXR: 11.498678, TimeStart: time.Date(2025, 2, 2, 8, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 2, 9, 0, 0, 0, loc)},
				{SEKPerkWh: 0.82296, EURPerkWh: 0.07157, EXR: 11.498678, TimeStart: time.Date(2025, 2, 2, 9, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 2, 10, 0, 0, 0, loc)},
				{SEKPerkWh: 0.78352, EURPerkWh: 0.06814, EXR: 11.498678, TimeStart: time.Date(2025, 2, 2, 10, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 2, 11, 0, 0, 0, loc)},
				{SEKPerkWh: 0.78628, EURPerkWh: 0.06838, EXR: 11.498678, TimeStart: time.Date(2025, 2, 2, 11, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 2, 12, 0, 0, 0, loc)},
				{SEKPerkWh: 0.74902, EURPerkWh: 0.06514, EXR: 11.498678, TimeStart: time.Date(2025, 2, 2, 12, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 2, 13, 0, 0, 0, loc)},
				{SEKPerkWh: 0.69015, EURPerkWh: 0.06002, EXR: 11.498678, TimeStart: time.Date(2025, 2, 2, 13, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 2, 14, 0, 0, 0, loc)},
				{SEKPerkWh: 0.70958, EURPerkWh: 0.06171, EXR: 11.498678, TimeStart: time.Date(2025, 2, 2, 14, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 2, 15, 0, 0, 0, loc)},
				{SEKPerkWh: 0.77892, EURPerkWh: 0.06774, EXR: 11.498678, TimeStart: time.Date(2025, 2, 2, 15, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 2, 16, 0, 0, 0, loc)},
				{SEKPerkWh: 0.91736, EURPerkWh: 0.07978, EXR: 11.498678, TimeStart: time.Date(2025, 2, 2, 16, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 2, 17, 0, 0, 0, loc)},
				{SEKPerkWh: 1.01292, EURPerkWh: 0.08809, EXR: 11.498678, TimeStart: time.Date(2025, 2, 2, 17, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 2, 18, 0, 0, 0, loc)},
				{SEKPerkWh: 1.06351, EURPerkWh: 0.09249, EXR: 11.498678, TimeStart: time.Date(2025, 2, 2, 18, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 2, 19, 0, 0, 0, loc)},
				{SEKPerkWh: 1.14251, EURPerkWh: 0.09936, EXR: 11.498678, TimeStart: time.Date(2025, 2, 2, 19, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 2, 20, 0, 0, 0, loc)},
				{SEKPerkWh: 0.99809, EURPerkWh: 0.0868, EXR: 11.498678, TimeStart: time.Date(2025, 2, 2, 20, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 2, 21, 0, 0, 0, loc)},
				{SEKPerkWh: 0.92553, EURPerkWh: 0.08049, EXR: 11.498678, TimeStart: time.Date(2025, 2, 2, 21, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 2, 22, 0, 0, 0, loc)},
				{SEKPerkWh: 0.59885, EURPerkWh: 0.05208, EXR: 11.498678, TimeStart: time.Date(2025, 2, 2, 22, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 2, 23, 0, 0, 0, loc)},
				{SEKPerkWh: 0.4634, EURPerkWh: 0.0403, EXR: 11.498678, TimeStart: time.Date(2025, 2, 2, 23, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 3, 0, 0, 0, 0, loc)},
			},
		},
		{
			name:     "test day 2",
			response: day2,
			wantErr:  false,
			wantPrices: Prices{
				{SEKPerkWh: 0.48455, EURPerkWh: 0.04214, EXR: 11.498678, TimeStart: time.Date(2025, 2, 3, 0, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 3, 1, 0, 0, 0, loc)},
				{SEKPerkWh: 0.40774, EURPerkWh: 0.03546, EXR: 11.498678, TimeStart: time.Date(2025, 2, 3, 1, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 3, 2, 0, 0, 0, loc)},
				{SEKPerkWh: 0.40809, EURPerkWh: 0.03549, EXR: 11.498678, TimeStart: time.Date(2025, 2, 3, 2, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 3, 3, 0, 0, 0, loc)},
				{SEKPerkWh: 0.40418, EURPerkWh: 0.03515, EXR: 11.498678, TimeStart: time.Date(2025, 2, 3, 3, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 3, 4, 0, 0, 0, loc)},
				{SEKPerkWh: 0.43545, EURPerkWh: 0.03787, EXR: 11.498678, TimeStart: time.Date(2025, 2, 3, 4, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 3, 5, 0, 0, 0, loc)},
				{SEKPerkWh: 0.59287, EURPerkWh: 0.05156, EXR: 11.498678, TimeStart: time.Date(2025, 2, 3, 5, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 3, 6, 0, 0, 0, loc)},
				{SEKPerkWh: 1.17815, EURPerkWh: 0.10246, EXR: 11.498678, TimeStart: time.Date(2025, 2, 3, 6, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 3, 7, 0, 0, 0, loc)},
				{SEKPerkWh: 1.71169, EURPerkWh: 0.14886, EXR: 11.498678, TimeStart: time.Date(2025, 2, 3, 7, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 3, 8, 0, 0, 0, loc)},
				{SEKPerkWh: 1.88912, EURPerkWh: 0.16429, EXR: 11.498678, TimeStart: time.Date(2025, 2, 3, 8, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 3, 9, 0, 0, 0, loc)},
				{SEKPerkWh: 1.72066, EURPerkWh: 0.14964, EXR: 11.498678, TimeStart: time.Date(2025, 2, 3, 9, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 3, 10, 0, 0, 0, loc)},
				{SEKPerkWh: 1.72101, EURPerkWh: 0.14967, EXR: 11.498678, TimeStart: time.Date(2025, 2, 3, 10, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 3, 11, 0, 0, 0, loc)},
				{SEKPerkWh: 1.45136, EURPerkWh: 0.12622, EXR: 11.498678, TimeStart: time.Date(2025, 2, 3, 11, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 3, 12, 0, 0, 0, loc)},
				{SEKPerkWh: 1.34891, EURPerkWh: 0.11731, EXR: 11.498678, TimeStart: time.Date(2025, 2, 3, 12, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 3, 13, 0, 0, 0, loc)},
				{SEKPerkWh: 1.32499, EURPerkWh: 0.11523, EXR: 11.498678, TimeStart: time.Date(2025, 2, 3, 13, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 3, 14, 0, 0, 0, loc)},
				{SEKPerkWh: 1.4294, EURPerkWh: 0.12431, EXR: 11.498678, TimeStart: time.Date(2025, 2, 3, 14, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 3, 15, 0, 0, 0, loc)},
				{SEKPerkWh: 1.67846, EURPerkWh: 0.14597, EXR: 11.498678, TimeStart: time.Date(2025, 2, 3, 15, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 3, 16, 0, 0, 0, loc)},
				{SEKPerkWh: 1.95225, EURPerkWh: 0.16978, EXR: 11.498678, TimeStart: time.Date(2025, 2, 3, 16, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 3, 17, 0, 0, 0, loc)},
				{SEKPerkWh: 2.47038, EURPerkWh: 0.21484, EXR: 11.498678, TimeStart: time.Date(2025, 2, 3, 17, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 3, 18, 0, 0, 0, loc)},
				{SEKPerkWh: 2.31319, EURPerkWh: 0.20117, EXR: 11.498678, TimeStart: time.Date(2025, 2, 3, 18, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 3, 19, 0, 0, 0, loc)},
				{SEKPerkWh: 2.30939, EURPerkWh: 0.20084, EXR: 11.498678, TimeStart: time.Date(2025, 2, 3, 19, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 3, 20, 0, 0, 0, loc)},
				{SEKPerkWh: 1.8882, EURPerkWh: 0.16421, EXR: 11.498678, TimeStart: time.Date(2025, 2, 3, 20, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 3, 21, 0, 0, 0, loc)},
				{SEKPerkWh: 1.49437, EURPerkWh: 0.12996, EXR: 11.498678, TimeStart: time.Date(2025, 2, 3, 21, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 3, 22, 0, 0, 0, loc)},
				{SEKPerkWh: 1.24347, EURPerkWh: 0.10814, EXR: 11.498678, TimeStart: time.Date(2025, 2, 3, 22, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 3, 23, 0, 0, 0, loc)},
				{SEKPerkWh: 0.61023, EURPerkWh: 0.05307, EXR: 11.498678, TimeStart: time.Date(2025, 2, 3, 23, 0, 0, 0, loc), TimeEnd: time.Date(2025, 2, 4, 0, 0, 0, 0, loc)},
			},
		},
		{
			name:     "invalid API response",
			response: "blah",
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, tc.response)
			}))
			defer ts.Close()
			pc := PriceClient{
				baseURL: ts.URL,
				client:  ts.Client(),
			}
			err := pc.LoadPrices()
			if (err != nil) != tc.wantErr {
				t.Errorf("LoadPrices() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if diff := cmp.Diff(tc.wantPrices, pc.prices); diff != "" {
				t.Errorf("LoadPrices() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPriceLoader(t *testing.T) {
	loc, err := time.LoadLocation(Locale)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, day2)
	}))
	defer ts.Close()
	fakec := fakeclock{
		curtime: time.Date(2025, 2, 2, 23, 59, 59, 0, loc),
	}
	clocksource = fakec.Now
	pc := PriceClient{
		baseURL: ts.URL,
		client:  ts.Client(),
	}
	done := make(chan struct{})

	// this should take around 2s to reload next days values, if it fails the timeout of 5s will fail the test
	go func() {
		pc.PriceLoader()
	}()

	// run a verification loop on the loaded prices
	go func() {
		for {
			time.Sleep(time.Millisecond * 500)
			pc.mutex.Lock()
			if len(pc.prices) > 0 {
				fmt.Println(pc.prices[0].SEKPerkWh)
				// First value for the newly loaded day should be 0.48455
				if pc.prices[0].SEKPerkWh == 0.48455 {
					pc.mutex.Unlock()
					break
				}
			}
			pc.mutex.Unlock()
		}
		close(done)
	}()

	// wait for either a timeout or a successful signal
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	select {
	case <-ctx.Done():
		t.Fatal("no loaded prices within the timeout, oh no!")
	case <-done:
		break
	}
}
