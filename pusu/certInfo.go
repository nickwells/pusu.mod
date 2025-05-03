package pusu

import (
	"crypto/tls"
	"crypto/x509"
	"log/slog"
	"os"
)

// CertInfo holds certificate information
type CertInfo struct {
	// parameters
	caCertFilename string // the file holding the CA's certificate
	certFilename   string // the file holding the program's certificate
	keyFilename    string // the file holding the program's private key

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
// certificate. Then it will add that to the certificate pool.
func (ci *CertInfo) PopulateCertPool(logger *slog.Logger) bool {
	// read the certificate file
	pem, err := os.ReadFile(ci.caCertFilename)
	if err != nil {
		logger.Error("couldn't read the certificate file",
			ErrorAttr(err),
			PemFileAttr(ci.caCertFilename),
		)

		return false
	}

	ci.certPool = x509.NewCertPool()

	// and add it to the certPool
	if ok := ci.certPool.AppendCertsFromPEM(pem); !ok {
		logger.Error("the PEM is invalid", PemFileAttr(ci.caCertFilename))

		return false
	}

	ci.certPoolPopulated = true

	return true
}

// PopulateCert will construct the program's certificate. It returns false if
// the certificate construction failed.
func (ci *CertInfo) PopulateCert(logger *slog.Logger) bool {
	var err error

	ci.cert, err = tls.LoadX509KeyPair(ci.certFilename, ci.keyFilename)
	if err != nil {
		logger.Error("couldn't load the x509 keypair",
			ErrorAttr(err),
			slog.String("certFileName", ci.certFilename),
			slog.String("keyFileName", ci.keyFilename),
		)

		return false
	}

	ci.certPopulated = true

	return true
}
