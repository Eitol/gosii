package gosii

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"net/http"
)

//go:embed zeus_sii.pem
var certBytes []byte

func buildHTTPClient() *http.Client {
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(certBytes)
	// add the default root CAs
	caCertPool, _ = x509.SystemCertPool()
	caCertPool.AppendCertsFromPEM(certBytes)
	var httpClient = &http.Client{
		Transport: &http.Transport{
			// Definir InsecureSkipVerify como true desactiva la verificación SSL
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				RootCAs:            caCertPool,
			},
		},
	}
	return httpClient
}
