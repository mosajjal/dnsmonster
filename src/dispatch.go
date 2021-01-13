package main

import (
	"sync"
)

func dispatchOutput(resultChannel chan DNSResult, exiting chan bool, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	for {
		select {
		case data := <-resultChannel:
			if stdoutOutputBool {
				stdoutResultChannel <- data
			}
			if fileOutputBool {
				fileResultChannel <- data
			}
			if clickhouseOutputBool {
				clickhouseResultChannel <- data
			}

		case <-exiting:
			return

		}
	}
}
