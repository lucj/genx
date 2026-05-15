package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// runReplay reads JSON-lines from filePath and sends each DataPoint through sink.
// In realtime mode it waits stepSeconds between sends and stamps the current time.
// In batch mode it sends all points immediately, preserving original timestamps.
// It stops early if ctx is cancelled.
func runReplay(ctx context.Context, filePath string, sink Sink, realtime bool, stepSeconds int) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("replay: cannot open file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	var ticker *time.Ticker
	if realtime {
		ticker = time.NewTicker(time.Duration(stepSeconds) * time.Second)
		defer ticker.Stop()
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var dp DataPoint
		if err := json.Unmarshal([]byte(line), &dp); err != nil {
			log.Printf("replay: skipping invalid line: %v", err)
			continue
		}
		if realtime {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
			}
			dp.Timestamp = time.Now().Unix()
		}
		if err := sink.Send(dp); err != nil {
			log.Printf("send error: %v", err)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("replay: read error: %v", err)
	}
	return nil
}
