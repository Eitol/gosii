package main

import (
	"errors"
	"fmt"
	"github.com/Eitol/gosii"
	"github.com/Eitol/gosii/pkg"
	"log"
	"time"
)

var latestTime *time.Time

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
	ssiClient := gosii.NewClient(&gosii.Opts{OnNewCaptcha: OnNewCaptcha})
	for rn := 12_000_000; rn < 13_000_000; rn++ {
		dv := pkg.GetRutDv(rn)
		rut := fmt.Sprintf("%d-%s", rn, dv)
		data, err := ssiClient.GetNameByRUT(rut)
		if err != nil {
			if errors.Is(err, gosii.ErrNotFound) {
				continue
			} else {
				log.Printf("Error: %s", err)
			}
		} else {
			log.Printf("Data: %s", data)
		}
	}
}
