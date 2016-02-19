package main

// httpoll asynchronously queries a provided set of URLs and presents
// them graphically in the browser, leveraging the delicious termui

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync"

	ui "github.com/gizak/termui"
)

const (
	bufferSize  = 50
	httpTimeout = 5.0
	refreshTime = 3.0
	rowHeight   = 8
)

// urlWidgets tracks all the widgets needed to present the state of a given
type urlWidgets struct {
	label     *ui.Par
	sparkline *ui.Sparkline
}

// histLock globally locks the history status map
// UI display should lock it for reading
// Web results should lock it for read and write.
var histLock sync.RWMutex

// statusString formats a string in termui color syntax
// given the provided 'ok' state (red if false, green if true)
func statusString(text string, ok bool) string {
	if ok {
		return fmt.Sprintf("[%s](fg-green)", text)
	} else {
		return fmt.Sprintf("[%s](fg-red)", text)
	}
}

// buildUI builds out the termUI user interface
// it returns a *urlWidgets which has a map of
// all widgets per domain so they can be accessed later.
func buildUI(urls []string) map[string]*urlWidgets {
	err := ui.Init()
	if err != nil {
		panic(err)
	}

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

	return widgets
}

func main() {
	urls := os.Args[1:]
	if len(urls) == 0 {
		fmt.Println("Error: No URLS to poll!")
		fmt.Println("Usage: httpoll http://firstdomain.com/foo https://seconddomain.com/bar")
		os.Exit(1)
	}

	for i, url := range urls {
		if !strings.HasPrefix(url, "http") {
			urls[i] = fmt.Sprintf("http://%s", url)
		}
	}

	statuses := make(chan SiteStatus)
	siteHistory := make(map[string]*SiteHistory)
	go fetch_urls(urls, siteHistory, statuses)

	widgets := buildUI(urls)
	defer ui.Close()

	ui.Handle("/sys/kbd/q", func(ui.Event) {
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/C-c", func(ui.Event) {
		ui.StopLoop()
	})
	ui.Handle("/timer/1s", func(ui.Event) {

		// Read the current state out of the history object
		// per URL, update the relevant widget contents
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
