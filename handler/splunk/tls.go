package splunk

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"strings"
)

const splunkCloudCACert = `
-----BEGIN CERTIFICATE-----
MIIB/DCCAaGgAwIBAgIBADAKBggqhkjOPQQDAjB+MSswKQYDVQQDEyJTcGx1bmsg
Q2xvdWQgQ2VydGlmaWNhdGUgQXV0aG9yaXR5MRYwFAYDVQQHEw1TYW4gRnJhbmNp
c2NvMRMwEQYDVQQKEwpTcGx1bmsgSW5jMQswCQYDVQQIEwJDQTEVMBMGA1UECxMM
U3BsdW5rIENsb3VkMB4XDTE0MTExMDA3MDAxOFoXDTM0MTEwNTA3MDAxOFowfjEr
MCkGA1UEAxMiU3BsdW5rIENsb3VkIENlcnRpZmljYXRlIEF1dGhvcml0eTEWMBQG
A1UEBxMNU2FuIEZyYW5jaXNjbzETMBEGA1UEChMKU3BsdW5rIEluYzELMAkGA1UE
CBMCQ0ExFTATBgNVBAsTDFNwbHVuayBDbG91ZDBZMBMGByqGSM49AgEGCCqGSM49
AwEHA0IABPRRy9i3yQcxgMpvCSsI7Qe6YZMimUHOecPZWaGz5jEfB4+p5wT7dF3e
QrgjDWshVJZvK6KGO7nDh97GnbVXrTCjEDAOMAwGA1UdEwQFMAMBAf8wCgYIKoZI
zj0EAwIDSQAwRgIhALMUgLYPtICN9ci/ZOoXeZxUhn3i4wIo2mPKEWX0IcfpAiEA
8Jid6bzwUqAdDZPSOtaEBXV9uRIrNua0Qxl1S55TlWY=
-----END CERTIFICATE-----
`

// CACerts is an x509 cert pool used when the Splunk API endpoint is using HTTPS, and has a certificate not recognized by the standard certificate authorities.
// You may add custom certificates to the pool, even after the handler has been instantiated.
var CACerts = x509.NewCertPool()

// TLSConfig is the config used by the splunk handler for unrecognized certificate authoritiess.
var TLSConfig = &tls.Config{RootCAs: CACerts}

func init() {
	CACerts.AppendCertsFromPEM([]byte(splunkCloudCACert))
}

// getHttpsClient returns a http.Client for communication with the Splunk API endpoint.
//
// It connects to the specified address using the standard TLS configuration, and if successful, returns the http.DefaultClient. If unsuccessful it returns a new http.Client using the CACerts cert pool.
func getHttpsClient(addr string) (*http.Client, error) {
	conn, err := tls.Dial("tcp", addr, nil)
	if err == nil {
		conn.Close()
		return http.DefaultClient, nil
	}

	return newHttpsClient(addr)
}

// newHttpsClient returns a http.Client using the CACerts cert pool.
// Additionally, if the address is a Splunk cloud address, it strips the "input-" prefix from the address when doing certificate validation. This is because Splunk cloud uses the cert for the web dashboard on the API address (which uses a different host name).
func newHttpsClient(addr string) (*http.Client, error) {
	var tlsConfig = *TLSConfig

	// splunk cloud uses the cert for prd-*.cloud.splunk.com on the hostname input-prd-*.
	// So here we're looking for splunk cloud, and faking the hostname so it validates
	host := strings.SplitN(addr, ":", 2)[0]
	if strings.HasSuffix(host, ".cloud.splunk.com") {
		tlsConfig.ServerName = strings.TrimPrefix(host, "input-")
	}

	httpsClient := &http.Client{Transport: &http.Transport{TLSClientConfig: &tlsConfig}}
	return httpsClient, nil
}
