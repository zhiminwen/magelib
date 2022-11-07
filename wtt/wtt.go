package wtt

import (
	"log"
	"time"
)

func WaitTillTrue(fn func() bool, timeout time.Duration, interval time.Duration) {
	start := time.Now()
	for {
		if fn() {
			return
		}
		if time.Since(start) > timeout {
			log.Fatalf("timeout while waiting for true")
		}
		log.Printf("sleep and retry for true condition...")
		time.Sleep(interval)
	}
}
