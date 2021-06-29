package main

import (
	"fmt"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
)

func sign() {
	jwt.NewWithClaims(jwt.SigningMethodHS256, &jwt.StandardClaims{
		Issuer: "bar",
	}).SignedString([]byte("shhhhh"))
}

func main() {
	// number of signing goroutines
	workerPool := 1

	// rate of signing requests per milliseconds
	milliRate := 500

	// duration of the test
	elapsed := 10 * time.Second

	workerChan := make(chan struct{})

	signed := make(chan struct{})

	totalSigned := 0
	go func() {
		for range signed {
			totalSigned++
		}
	}()
	requested := make(chan struct{})
	totalSent := 0
	go func() {
		for range requested {
			totalSent++
		}
	}()
	for i := 1; i <= workerPool; i++ {
		go func() {
			for range workerChan {
				sign()
				signed <- struct{}{}
			}
		}()
	}
	end := time.After(elapsed)
	for range time.Tick(1 * time.Millisecond) {
		select {
		case <-end:
			fmt.Printf("signed %v times in %v seconds, requested %v signatures\n", totalSigned, elapsed, totalSent)
			os.Exit(0)
		default:
			for i := 0; i < milliRate; i++ {
				requested <- struct{}{}
				workerChan <- struct{}{}
			}
		}
	}
}
