package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"sync"
)

func main() {
	url := flag.String("url", "", "URL tujuan")
	token := flag.String("token", "", "access token")
	body := flag.String("body", "{}", "body JSON")
	file := flag.String("file", "", "berkas berisi body JSON (menggantikan -body)")
	n := flag.Int("n", 10, "jumlah request serentak")
	flag.Parse()

	if *file != "" {
		b, err := os.ReadFile(*file)
		if err != nil {
			fmt.Println("gagal membaca berkas:", err)
			os.Exit(1)
		}
		*body = string(b)
	}

	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan string, *n)

	for i := 0; i < *n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			req, _ := http.NewRequest(http.MethodPost, *url, bytes.NewBufferString(*body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+*token)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				results <- "error: " + err.Error()
				return
			}
			defer resp.Body.Close()
			_, _ = io.Copy(io.Discard, resp.Body)
			results <- resp.Status
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	count := map[string]int{}
	for r := range results {
		count[r]++
	}
	keys := make([]string, 0, len(count))
	for k := range count {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("%3d x %s\n", count[k], k)
	}
}
