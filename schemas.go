package gosii

type CaptchaResp struct {
	TxtCaptcha string `json:"txtCaptcha"`
}

type CommercialActivity struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type Citizen struct {
	Rut        string               `json:"rut"`
	Run        string               `json:"run"`
	Name       string               `json:"name"`
	Activities []CommercialActivity `json:"activities"`
}
