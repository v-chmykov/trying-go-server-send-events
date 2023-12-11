package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

//go:embed index.html
var indexHTML []byte
var r *rand.Rand

func init() {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/data-source", dataSourceHandler)

	http.ListenAndServe(":8888", nil)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write(indexHTML)
}

func dataSourceHandler(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")

	dataCh := make(chan int)

	go generateRandomData(r.Context(), dataCh)

	for data := range dataCh {
		event, err := formatEvent("data-update", data)
		if err != nil {
			fmt.Println(err)
			break
		}

		_, err = fmt.Fprint(w, event)
		if err != nil {
			fmt.Println(err)
			break
		}

		flusher.Flush()
	}
}

func generateRandomData(ctx context.Context, dataCh chan<- int) {
	ticker := time.NewTicker(3 * time.Second)

outerLoop:
	for {
		select {
		case <-ctx.Done():
			break outerLoop
		case <-ticker.C:
			dataCh <- r.Intn(100)
		}
	}

	ticker.Stop()

	close(dataCh)
}

func formatEvent(event string, data any) (string, error) {
	m := map[string]any{
		"data": data,
	}

	buff := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(buff)

	err := encoder.Encode(m)
	if err != nil {
		return "", err
	}

	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf("event: %s\n", event))
	sb.WriteString(fmt.Sprintf("data: %v\n\n", buff.String()))

	return sb.String(), nil
}
