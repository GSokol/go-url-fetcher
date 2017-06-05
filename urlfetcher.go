//
// urlfetcher.go
// Copyright (C) 2017 Grigorii Sokolik <g.sokol99@g-sokol.info>
//
// Distributed under terms of the MIT license.
//

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
)

const (
	MAX_ACTIVE_FETCHERS  = 5
	MAX_WAITING_FETCHERS = 1
	SEARCHED_STRING      = "Go"
	URL_OUT_TPL          = "Count for %s: %d\n\n"
	TOTAL_OUT_TPL        = "Total: %d\n"
)

type result struct {
	Url   string
	Count int
}

func fetchUrl(url string) (res result) {
	res.Url = url
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return
	}
	data, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Println(err)
		return
	}

	res.Count = len(strings.Split(string(data), SEARCHED_STRING)) - 1
	return
}

func fetchUrls(urlC chan string, results chan result, l sync.Locker, af *int, bf *int, wg *sync.WaitGroup) {
	l.Lock()
	*af++
	l.Unlock()
	for url := range urlC {
		l.Lock()
		*bf++
		l.Unlock()
		results <- fetchUrl(url)
		l.Lock()
		*bf--
		if *bf < *af-MAX_WAITING_FETCHERS {
			*af--
			l.Unlock()
			break
		}
		l.Unlock()
	}
	wg.Done()
}

func balanceUrl(urlC chan string, results chan result) {
	activeFetchers := 0
	busyFetchers := 0
	fetchersMutex := &sync.RWMutex{}
	wg := &sync.WaitGroup{}
	balansedUrlC := make(chan string)
	for url := range urlC {
		fetchersMutex.RLock()
		if activeFetchers < MAX_ACTIVE_FETCHERS && busyFetchers == activeFetchers {
			wg.Add(1)
			go fetchUrls(balansedUrlC, results, fetchersMutex, &activeFetchers, &busyFetchers, wg)
		}
		fetchersMutex.RUnlock()
		balansedUrlC <- url
	}
	close(balansedUrlC)

	wg.Wait()
	close(results)
}

func collectResults(buffer *[]result, total *int, results chan result, l sync.Locker) {
	l.Lock()
	defer l.Unlock()

	for r := range results {
		*buffer = append(*buffer, r)
		*total += r.Count
	}
}

func printResults(results []result, total int) {
	for _, r := range results {
		fmt.Printf(URL_OUT_TPL, r.Url, r.Count)
	}
	fmt.Printf(TOTAL_OUT_TPL, total)
}

func main() {
	var url string
	urlC := make(chan string)
	buffer := []result{}
	total := 0
	results := make(chan result)
	rLock := &sync.RWMutex{}
	go collectResults(&buffer, &total, results, rLock)
	go balanceUrl(urlC, results)
	for {
		_, err := fmt.Scanf("%s\n", &url)
		if err == io.EOF {
			break
		}
		urlC <- url
	}
	close(urlC)
	rLock.RLock()
	defer rLock.RUnlock()
	printResults(buffer, total)
}
