package main

import (
	"net/http"
	"time"
)

// SiteStatus contains the information about a poll of a website
// url: url queried
// status: any status messages (from remote server, like '200 OK', or remote like "can't look up host")
// healthy: bool indicating if it worked or not. false if either local or remote errors occur
// responseMillis: response time in milliseconds. 0 in the event of an error.
type SiteStatus struct {
	url            string
	status         string
	healthy        bool
	responseMillis int
}

// SiteHistory contains the most recent poll data, as well as the history of the
// requests over time.
type SiteHistory struct {
	now        SiteStatus
	pastMillis []int
}

// fetch_url uses an existing http.Client (which can be configured
// with timeout, etc if need be) to time the fetch of a single url,
// returning the results to the 'statuses' channel
func fetch_url(url string, statuses chan SiteStatus, c *http.Client) {
	time_start := time.Now()
	resp, err := c.Get(url)
	if err != nil {
		statuses <- SiteStatus{url, err.Error(), false, 0}
	} else {
		defer resp.Body.Close()
		elapsed := int(time.Since(time_start) / time.Millisecond)
		if resp.StatusCode < 300 {
			statuses <- SiteStatus{url, resp.Status, true, elapsed}
		} else {
			statuses <- SiteStatus{url, resp.Status, false, 0}
		}
	}
}

// time_response is a wrapper method which sets up the initial client
// and polling interval, handling executing the requests periodically
// as needed.
func time_response(url string, statuses chan SiteStatus) {
	timeout := time.Duration(httpTimeout * time.Second)
	client := http.Client{Timeout: (timeout * time.Second)}
	ticker := time.NewTicker(refreshTime * time.Second)
	for {
		select {
		case <-ticker.C:
			fetch_url(url, statuses, &client)
		}
	}
}

// fetch_urls  takes a list of URL's and a site history map
// It sets up the starting state of the map and launches polling goroutines
// on all of the provided URLs
func fetch_urls(urls []string, siteHistory map[string]*SiteHistory, statuses chan SiteStatus) {
	for _, url := range urls {
		histLock.Lock()
		hist := SiteHistory{
			pastMillis: make([]int, bufferSize, bufferSize),
		}
		siteHistory[url] = &hist
		histLock.Unlock()
		go time_response(url, statuses)
	}
	for {
		select {
		case s := <-statuses:
			histLock.Lock()
			hist := siteHistory[s.url]
			hist.now = s
			hist.pastMillis = append(hist.pastMillis[1:], s.responseMillis)
			histLock.Unlock()
		}
	}
}

// upAverage provides the average of a slice of integers after removing
// all of the 0 values (0 are a placeholder for down.)
// when the slice has no non-zero values it should return 0.0
func upAverage(data []int) float64 {
	total := 0
	countable := 0
	for _, value := range data {
		if value > 0 {
			total += value
			countable++
		}
	}
	if countable > 0 {
		return float64(total) / float64(countable)
	} else {
		return 0.0
	}
}
