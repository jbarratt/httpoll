package main

import (
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Test the HTTP fetcher method to ensure it handles errors properly

// timeout := time.Duration(httpTimeout * time.Second)
// client := http.Client{Timeout: timeout}
// func fetch_url(url string, statuses chan SiteStatus, c *http.Client) {

func TestBadStatus(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "I said GO AWAY.")
	}))
	defer ts.Close()

	status := make(chan SiteStatus)
	client := http.Client{Timeout: time.Second}
	go fetch_url(ts.URL, status, &client)
	rv := <-status
	if rv.healthy {
		t.Errorf("This status should not be healthy")
	}
	if rv.responseMillis != 0 {
		t.Errorf("A bad response should count for zero time")
	}
	if !strings.HasPrefix(rv.status, "404") {
		t.Errorf("response was fatal for the wrong reason: %s", rv.status)
	}
}

func TestSlowServer(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Worth the wait?")
	}))
	defer ts.Close()

	status := make(chan SiteStatus)
	client := http.Client{Timeout: (1 * time.Millisecond)}
	go fetch_url(ts.URL, status, &client)
	rv := <-status
	if rv.healthy {
		t.Errorf("This status should not be healthy")
	}
	if rv.responseMillis != 0 {
		t.Errorf("A bad response should count for zero time")
	}
}

func TestHealthyServer(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "So nice to see you.")
	}))
	defer ts.Close()

	status := make(chan SiteStatus)
	client := http.Client{Timeout: time.Second}
	go fetch_url(ts.URL, status, &client)
	rv := <-status
	if !rv.healthy {
		t.Errorf("This status should be healthy: %s", rv.status)
	}

}

func TestUpAverageAllZero(t *testing.T) {
	avg := upAverage([]int{0, 0, 0, 0, 0})
	if avg > 0.000001 {
		t.Errorf("Average should be close to zero, not %0.2f", avg)
	}
}

func TestUpAverageSomeZero(t *testing.T) {
	avg := upAverage([]int{10, 0, 0, 0, 10})
	if math.Abs(avg-10.0) > 0.000001 {
		t.Errorf("Average should be very close to 10, not %0.2f", avg)
	}
}
