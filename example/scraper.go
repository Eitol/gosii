package main

import (
	"errors"
	"fmt"
	"github.com/Eitol/gosii"
	"github.com/Eitol/gosii/pkg"
	"github.com/mailru/easyjson"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

var latestTime *time.Time

const idxFileName = "last_run_idx.txt"
const outDir = "output"
const filesPerDir = 10_000

func saveLastRun(run int) {
	err := os.WriteFile(idxFileName, []byte(fmt.Sprintf("%d", run)), 0644)
	if err != nil {
		log.Fatalf("Error saving last run: %s", err)
	}
}

func readLastRun() int {
	data, err := os.ReadFile(idxFileName)
	if err != nil {
		log.Printf("Error reading last run: %s", err)
		return 0
	}
	run, err := strconv.Atoi(string(data))
	if err != nil {
		log.Fatalf("Error converting last run: %s", err)
	}
	return run
}

func OnNewCaptcha(captcha *gosii.Captcha) {
	if latestTime == nil {
		log.Printf("First captcha: %s - %s", captcha.Solution, captcha.Text)
	} else {
		log.Printf("Renew captcha: %s - %s", captcha.Solution, captcha.Text)
		passedTime := time.Since(*latestTime)
		log.Printf("Passed time: %s", passedTime)
	}
	nowTime := time.Now()
	latestTime = &nowTime
}

func main() {
	nworkers := 10
	startRUT := 1
	endRUT := 30_000_000
	lastRun := readLastRun()
	if lastRun > startRUT {
		startRUT = lastRun
	}
	jobChan := make(chan int, nworkers)
	mutex := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	wg.Add(nworkers)
	go func() {
		for i := startRUT; i < endRUT; i++ {
			jobChan <- i
		}
		close(jobChan)
	}()
	err := os.MkdirAll(outDir, 0755)
	if err != nil {
		log.Fatalf("Error creating output dir: %s", err)
	}
	for i := 0; i < nworkers; i++ {
		go func() {
			defer wg.Done()
			ssiClient := gosii.NewClient(&gosii.Opts{OnNewCaptcha: OnNewCaptcha})
			for run := range jobChan {
				dv := pkg.GetRutDv(run)
				rut := fmt.Sprintf("%d-%s", run, dv)
				data, err := ssiClient.GetNameByRUT(rut)
				if err != nil {
					if errors.Is(err, gosii.ErrNotFound) {
						continue
					} else {
						log.Printf("Error: %s", err)
					}
				} else {
					mutex.Lock()
					saveLastRun(run)
					saveOutput(data)
					mutex.Unlock()
					log.Printf("Found: %s: %s", rut, data)
				}
			}
		}()
	}
	wg.Wait()
}

func saveOutput(output *gosii.Citizen) {
	fileName := buildOutFileName(output)
	outJson, err := easyjson.Marshal(output)
	if err != nil {
		log.Fatalf("Error marshaling output: %s", err)
	}
	err = os.WriteFile(fileName, outJson, 0644)
	if err != nil {
		log.Fatalf("Error saving output: %s", err)
	}
}

func buildOutFileName(output *gosii.Citizen) string {
	run := output.Rut[:len(output.Rut)-2]
	runInt, err := strconv.Atoi(run)
	if err != nil {
		log.Fatalf("Error converting run: %s", err)
	}
	subDir := fmt.Sprintf("%d", runInt/filesPerDir)
	err = os.MkdirAll(filepath.Join(outDir, subDir), 0755)
	if err != nil {
		log.Fatalf("Error creating output dir: %s", err)
	}
	return filepath.Join(outDir, subDir, fmt.Sprintf("%s.json", output.Rut))
}
