package gosii

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	XpathRazonSocial = "html body div div:nth-child(4)"
	XpathActivities  = "html body div table tr"

	siiNameByRUTURL = "https://zeus.sii.cl/cvc_cgi/stc/getstc"
)

var ErrNotFound = errors.New("not found")
var ErrCaptcha = errors.New("not found")

type siiHTTPClient struct {
	captcha      *Captcha
	captchaMutex sync.Mutex
	opts         Opts
}

type Opts struct {
	OnNewCaptcha func(captcha *Captcha)
}

func NewClient(opts *Opts) Client {
	if opts == nil {
		opts = &Opts{}
	}
	return &siiHTTPClient{opts: *opts}
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
func (c *siiHTTPClient) GetNameByRUT(rut string) (*Citizen, error) {
	err := c.assertCaptcha()
	if err != nil {
		return nil, err
	}
	v, err := c.getUserByRUTAndCaptcha(rut, *c.captcha)
	if err != nil {
		if errors.Is(err, ErrCaptcha) {
			c.captcha = nil
			return c.GetNameByRUT(rut)
		}
	}
	return v, err
}

func (c *siiHTTPClient) assertCaptcha() error {
	c.captchaMutex.Lock()
	if c.captcha == nil {
		newCaptcha, err := fetchCaptcha()
		if err != nil {
			return err
		}
		if c.opts.OnNewCaptcha != nil {
			c.opts.OnNewCaptcha(newCaptcha)
		}
		c.captcha = newCaptcha
	}
	c.captchaMutex.Unlock()
	return nil
}

// fetchCaptcha fetches a captcha from the SII's service.
func (c *siiHTTPClient) getUserByRUTAndCaptcha(rut string, captcha Captcha) (*Citizen, error) {
	req, err := c.buildRequest(rut, captcha)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return c.parseSIIHTMLResponse(string(body))
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
		c.captcha = nil
		return nil, ErrCaptcha
	}

	razonSocial := strings.TrimSpace(doc.Find(XpathRazonSocial).Text())
	if razonSocial == "" || razonSocial == "**" {
		return nil, ErrNotFound
	}
	var actividades []CommercialActivity

	doc.Find(XpathActivities).Each(func(i int, s *goquery.Selection) {
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
