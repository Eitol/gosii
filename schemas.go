package sii

type CaptchaResp struct {
	TxtCaptcha string `json:"txtCaptcha"`
}

type CommercialActivity struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type Citizen struct {
	Rut        string               `json:"rut"`
	Name       string               `json:"name"`
	Activities []CommercialActivity `json:"activities"`
}
