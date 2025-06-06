package pusu

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// CertInfo holds certificate information
type CertInfo struct {
	// parameters
	CACertFilename string // the file holding the CA's certificate
	CertFilename   string // the file holding the program's certificate
	KeyFilename    string // the file holding the program's private key

	// program data
	cert     tls.Certificate // the program's certificate
	certPool *x509.CertPool  // the program's certificate pool

	certPopulated     bool // set to true only if successfully populated
	certPoolPopulated bool // set to true only if successfully populated
}

// Cert returns the certificate
func (ci CertInfo) Cert() tls.Certificate {
	if !ci.certPopulated {
		panic("the cert has not been successfully populated")
	}

	return ci.cert
}

// CertPool returns a pointer to the certificate pool
func (ci CertInfo) CertPool() *x509.CertPool {
	if !ci.certPoolPopulated {
		panic("the certPool has not been successfully populated")
	}

	return ci.certPool
}

// PopulateCertPool will first read the contents of the certification
// authority's certificate file, which should contain a PEM-coded
// certificate. Then it will add that to the certificate pool. It will return
// a non-nil error if the certificate file cannot be read or the certificate
// cannot be added to the certificate pool.
func (ci *CertInfo) PopulateCertPool() error {
	// read the certificate file
	pem, err := os.ReadFile(ci.CACertFilename)
	if err != nil {
		return fmt.Errorf("couldn't read the CA certificate file: %q: %w",
			ci.CACertFilename, err)
	}

	ci.certPool = x509.NewCertPool()

	// and add it to the certPool
	if ok := ci.certPool.AppendCertsFromPEM(pem); !ok {
		return fmt.Errorf("the PEM is invalid: %q", ci.CACertFilename)
	}

	ci.certPoolPopulated = true

	return nil
}

// PopulateCert will construct the program's certificate. It will return
// a non-nil error if the certificate cannot be loaded
func (ci *CertInfo) PopulateCert() error {
	var err error

	ci.cert, err = tls.LoadX509KeyPair(ci.CertFilename, ci.KeyFilename)
	if err != nil {
		return fmt.Errorf(
			"couldn't load x509 keypair: certFile: %q, keyFile: %q, err: %w",
			ci.CertFilename, ci.KeyFilename,
			err)
	}

	ci.certPopulated = true

	return nil
}
