package gosii

import (
	_ "embed"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	xpathRazonSocial = "html body div div:nth-child(4)"
	xpathActivities  = "html body div table tr"

	siiNameByRUTURL = "https://zeus.sii.cl/cvc_cgi/stc/getstc"
)

var ErrNotFound = errors.New("not found")
var ErrCaptcha = errors.New("not found")

type siiHTTPClient struct {
	captcha      *Captcha
	captchaMutex sync.Mutex
	opts         Opts
	httpClient   *http.Client
	requestCount atomic.Uint64
}

type Opts struct {
	OnNewCaptcha func(captcha *Captcha)
}

func NewClient(opts *Opts) Client {
	httpClient := buildHTTPClient()

	if opts == nil {
		opts = &Opts{}
	}
	return &siiHTTPClient{opts: *opts, httpClient: httpClient}
}

// GetNameByRUT fetches the name of a citizen from the Servicio de Impuestos Internos (SII)
// of Chile given the RUT (Rol Único Tributario), a unique tax number of the citizen.
//
// The method fetches (and resolves) a captcha first, which is necessary to make the request to SII's service.
// Then, it uses the provided RUT and the fetched captcha to get the information of the citizen.
// The RUT must be provided in the format of "12345678-9" or "12.345.678-9" or "123456789"
//
// This method first makes a POST request to the SII's service with the RUT and the captcha,
// then parses the HTML response to extract the citizen's name.
//
// The method returns a pointer to a Citizen structure, which contains the name of the citizen
// and the commercial activities associated with the citizen, or an error if the request fails
// or the RUT is not found.
//
// Response example: Citizen{Name:"MIGUEL JUAN SEBASTIAN PINERA ECHENIQUE", Activities:[]string{"829900"}}
//
// Returns sii.ErrNotFound if the RUT is not found.
//
// Please note that this method relies on the structure of SII's service and its response.
// If the service URL or the response structure changes, this method may not work as expected.
func (c *siiHTTPClient) GetNameByRUT(rut string) (*Citizen, *RequestMetadata, error) {
	captcha, err := c.assertCaptcha()
	if err != nil {
		return nil, nil, err
	}
	citizen, meta, err := c.getUserByRUTAndCaptcha(rut, *captcha)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil, ErrNotFound
		}
		if errors.Is(err, ErrCaptcha) {
			c.captchaMutex.Lock()
			if c.captcha != nil && c.captcha.Text == captcha.Text {
				c.captcha = nil
			}
			c.captchaMutex.Unlock()
			return c.GetNameByRUT(rut)
		}
	}
	return citizen, &meta, err
}

func (c *siiHTTPClient) assertCaptcha() (*Captcha, error) {
	c.captchaMutex.Lock()
	defer c.captchaMutex.Unlock()
	if c.captcha == nil {
		newCaptcha, err := c.fetchCaptcha()
		if err != nil {
			return nil, err
		}
		c.captcha = newCaptcha
		if c.opts.OnNewCaptcha != nil {
			c.opts.OnNewCaptcha(newCaptcha)
		}
	}
	return &Captcha{
		Text:     c.captcha.Text,
		Solution: c.captcha.Solution,
	}, nil
}

// fetchCaptcha fetches a captcha from the SII's service.
func (c *siiHTTPClient) getUserByRUTAndCaptcha(rut string, captcha Captcha) (*Citizen, RequestMetadata, error) {
	attempts := 3
	var err error
	var body []byte
	var requestTimes []time.Duration
	for attempts > 0 {
		// time btwn 0 and 8 seconds
		awaitSecondsTime := time.Duration(rand.Intn(8)) * time.Second
		var req *http.Request
		req, err = c.buildRequest(rut, captcha)
		if err != nil {
			break
		}
		var res *http.Response
		c.requestCount.Add(1)
		startTime := time.Now()
		res, err = c.httpClient.Do(req)
		endTime := time.Since(startTime)
		requestTimes = append(requestTimes, endTime)
		if err != nil {
			attempts--
			time.Sleep(awaitSecondsTime)
			continue
		}
		body, err = io.ReadAll(res.Body)
		_ = res.Body.Close()
		if err != nil {
			attempts--
			time.Sleep(awaitSecondsTime)
			continue
		}
		break
	}
	avgTime := float64(0)
	for _, t := range requestTimes {
		avgTime += float64(t)
	}
	avgTime = avgTime / float64(len(requestTimes))
	meta := RequestMetadata{
		TotalCount: int(c.requestCount.Load()),
		AvgTime:    avgTime,
		Attempts:   3 - attempts,
	}
	if err != nil {
		return nil, meta, err
	}
	ctz, err := c.parseSIIHTMLResponse(string(body))
	if err != nil {
		if strings.Contains(string(body), "**") {
			return nil, meta, ErrNotFound
		}
		return nil, meta, err
	}
	ctz.Rut = rut
	return ctz, meta, nil
}

func (c *siiHTTPClient) buildRequest(rut string, captcha Captcha) (*http.Request, error) {
	rut = strings.ReplaceAll(rut, ".", "")
	rut = strings.ReplaceAll(rut, "-", "")
	run := rut[:len(rut)-1]
	dv := strings.ToUpper(rut[len(rut)-1:])
	url := siiNameByRUTURL
	method := "POST"
	payloadStr := "RUT=" + run +
		"&DV=" + dv +
		"&txt_captcha=" + captcha.Text +
		"&txt_code=" + captcha.Solution +
		"&PRG=STC" +
		"&OPC=NOR"
	payload := strings.NewReader(payloadStr)
	return http.NewRequest(method, url, payload)
}

func (c *siiHTTPClient) parseSIIHTMLResponse(html string) (*Citizen, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	if strings.Contains(html, "Por favor reingrese Captcha") {
		return nil, ErrCaptcha
	}

	razonSocial := strings.TrimSpace(doc.Find(xpathRazonSocial).Text())
	if razonSocial == "" || razonSocial == "**" {
		return nil, ErrNotFound
	}
	var actividades []CommercialActivity

	doc.Find(xpathActivities).Each(func(i int, s *goquery.Selection) {
		if i > 0 {
			codigo := s.Find("td:nth-child(2) font").Text()
			var codeInt int
			codeInt, err = strconv.Atoi(codigo)
			if err != nil {
				return
			}
			if codeInt > 1970 && codeInt <= time.Now().Year() {
				return
			}
			actividades = append(actividades, CommercialActivity{
				Code: codigo,
			})
		}
	})

	return &Citizen{
		Name:       razonSocial,
		Activities: actividades,
	}, nil
}
