package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	ui "github.com/gizak/termui"
)

const (
	bufferSize  = 50
	httpTimeout = 5.0
	refreshTime = 3.0
	rowHeight   = 8
)

type SiteStatus struct {
	url            string
	status         string
	healthy        bool
	responseMillis int
}

type SiteHistory struct {
	now        SiteStatus
	pastMillis []int
}

type urlWidgets struct {
	label     *ui.Par
	sparkline *ui.Sparkline
}

var histLock sync.RWMutex

func (s SiteStatus) String() string {
	return fmt.Sprintf("{url: %s, status: %s, healthy? %s, responseTime: %d ms}", s.url, s.status, s.healthy, s.responseMillis)
}

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

func time_response(url string, statuses chan SiteStatus) {
	timeout := time.Duration(httpTimeout * time.Second)
	client := http.Client{Timeout: timeout}
	ticker := time.NewTicker(refreshTime * time.Second)
	for {
		select {
		case <-ticker.C:
			fetch_url(url, statuses, &client)
		}
	}
}

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

func statusString(text string, ok bool) string {
	if ok {
		return fmt.Sprintf("[%s](fg-green)", text)
	} else {
		return fmt.Sprintf("[%s](fg-red)", text)
	}
}

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

func main() {
	statuses := make(chan SiteStatus)
	siteHistory := make(map[string]*SiteHistory)
	urls := os.Args[1:]
	go fetch_urls(urls, siteHistory, statuses)

	err := ui.Init()
	if err != nil {
		panic(err)
	}
	defer ui.Close()

	widgets := make(map[string]*urlWidgets)
	for _, url := range urls {

		// Text panel
		p := ui.NewPar(url)
		p.Height = rowHeight
		p.TextFgColor = ui.ColorWhite

		// sparkline
		s := ui.NewSparkline()
		s.Data = make([]int, bufferSize, bufferSize)
		s.LineColor = ui.ColorGreen
		s.Height = rowHeight - 1

		// UI container for sparkline
		spls := ui.NewSparklines(s)
		spls.Height = rowHeight
		spls.Border = false

		widgets[url] = &urlWidgets{
			label:     p,
			sparkline: &s,
		}
		ui.Body.AddRows(ui.NewRow(
			ui.NewCol(6, 0, p),
			ui.NewCol(6, 0, spls),
		))
	}

	ui.Body.Align()
	ui.Render(ui.Body)

	ui.Handle("/sys/kbd/q", func(ui.Event) {
		ui.StopLoop()
	})
	ui.Handle("/timer/1s", func(ui.Event) {
		histLock.RLock()
		for _, url := range urls {
			state := siteHistory[url]

			// build the display string
			var buffer bytes.Buffer

			// domain & timing
			buffer.WriteString(fmt.Sprintf("%s: %d ms", url,
				state.now.responseMillis))
			buffer.WriteString("\n")

			// historical performance
			buffer.WriteString(fmt.Sprintf("Average performance: %0.2f ms",
				upAverage(state.pastMillis)))
			buffer.WriteString("\n")

			// overall status string
			buffer.WriteString(statusString(state.now.status, state.now.healthy))
			widgets[url].label.Text = buffer.String()

			// load the sparkline data
			sl := widgets[url].sparkline
			for i := 0; i < bufferSize; i++ {
				sl.Data[i] = state.pastMillis[i]
			}
		}
		histLock.RUnlock()
		ui.Body.Align()
		ui.Render(ui.Body)

	})
	ui.Loop()
}
