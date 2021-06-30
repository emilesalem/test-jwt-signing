package main

import (
	"context"
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

	// duration of the test
	elapsed := 10 * time.Millisecond

	deadline := time.Now().Add(elapsed)

	// number of signing goroutines
	workerPool := 1

	// rate of signing requests per milliseconds
	milliRate := 1

	workerChan := make(chan chan struct{})

	signed := make(chan struct{})

	available := make(chan struct{})

	dispatch := make(chan string)

	totalSigned := 0

	mainCtx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	dispatchCtx, dispatchCancel := context.WithDeadline(mainCtx, deadline)
	go func() {
		defer dispatchCancel()
		requestNb := 1
		workChanNb := 0
		for {
			select {
			case x := <-workerChan:
				go func() {
					for {
						select {
						case <-x:
							<-available
							dispatch <- fmt.Sprintf(`workchan: %v requestNb: %v`, workChanNb, requestNb)
							requestNb++
						case <-dispatchCtx.Done():
							return
						}
					}
				}()
				workChanNb++
			case <-dispatchCtx.Done():
				println(`done dispatching`)
				return
			}
		}
	}()

	counterCtx, counterCancel := context.WithDeadline(mainCtx, deadline)
	go func() {
		defer counterCancel()
		for {
			select {
			case <-signed:
				totalSigned++
			case <-counterCtx.Done():
				println(`we're done`)
				cancel()
				return
			}
		}
	}()

	for i := 1; i <= workerPool; i++ {
		wCtx, wCancel := context.WithDeadline(mainCtx, deadline)
		go func() {
			defer wCancel()
			available <- struct{}{}
			for {
				select {
				case x := <-dispatch:
					fmt.Printf("signing request %v\n", x)
					sign()
					signed <- struct{}{}
					available <- struct{}{}
				case <-wCtx.Done():
					close(available)
					close(signed)
					println(`we're done 2`)
					return
				}
			}
		}()
	}

	totalSent := 0
	for range time.Tick(1 * time.Millisecond) {
		select {
		case <-mainCtx.Done():
			close(workerChan)
			fmt.Printf("signed %v times in %v seconds, requested %v signatures\n", totalSigned, elapsed, totalSent)
			os.Exit(0)
		default:
			workChan := make(chan struct{})
			workerChan <- workChan
			go func() {
				for i := 0; i < milliRate; i++ {
					totalSent++
					workChan <- struct{}{}
				}
				close(workChan)
			}()
		}
	}
}
