package sii

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// siiCaptchaURL is the URL from where the CAPTCHA is fetched.
var siiCaptchaURL = "https://zeus.sii.cl/cvc_cgi/stc/CViewCaptcha.cgi"

type Captcha struct {
	Text     string `json:"text"`
	Solution string `json:"solution"`
}

// fetchCaptcha is responsible for retrieving a captcha challenge from the SII's service.
//
// The method makes a POST request to the SII's captcha service with the "oper=0" payload,
// which instructs the service to generate a new captcha.
//
// The response from the service is expected to be a JSON object which includes an attribute
// "TxtCaptcha", a Base64 encoded string which represents the captcha image. The captcha text
// (i.e., the solution to the captcha) is embedded in the Base64 encoded string.
//
// This method then decodes the Base64 encoded string to retrieve the captcha text. Specifically,
// the method slices the decoded string from the 36th to 40th byte, as this slice is observed to
// represent the captcha text. This observation is based on the assumption that the structure of
// the Base64 encoded string remains consistent.
//
// The method returns a pointer to a Captcha structure, which contains the Base64 encoded string
// and the decoded captcha text, or an error if any step of the request or processing fails.
//
// Please note that this method relies on the structure of SII's captcha service and its response.
// If the service URL or the response structure changes, this method may not work as expected.
func fetchCaptcha() (*Captcha, error) {
	resp, err := http.DefaultClient.Post(
		siiCaptchaURL, "application/json", strings.NewReader("oper=0"),
	)
	if err != nil {
		return nil, err
	}
	captchaResp := CaptchaResp{}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(respBody, &captchaResp)
	if err != nil {
		return nil, err
	}
	txtCaptcha := captchaResp.TxtCaptcha
	return solveCaptcha(err, txtCaptcha)
}

func solveCaptcha(err error, txtCaptcha string) (*Captcha, error) {
	code, err := base64.StdEncoding.DecodeString(txtCaptcha)
	if err != nil {
		return nil, err
	}
	code = code[36:40]

	return &Captcha{Text: txtCaptcha, Solution: string(code)}, nil
}