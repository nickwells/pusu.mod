package pusu

import (
	"github.com/nickwells/filecheck.mod/filecheck"
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/psetter"
)

const (
	certInfoParamNameCACertFilename = "ca-cert-filename"
	certInfoParamNameCertFilename   = "cert-filename"
	certInfoParamNameKeyFilename    = "key-filename"
)

// AddParams adds the parameters for the certificate info. The standard
// parameter names will be prefixed by the supplied prefix string (which
// should end with a dash); this is to allow multiple sets of parameters to
// be given
func (ci *CertInfo) AddParams(prefix string) param.PSetOptFunc {
	return func(ps *param.PSet) error {
		ps.Add(prefix+certInfoParamNameCACertFilename,
			psetter.Pathname{
				Value:       &ci.caCertFilename,
				Expectation: filecheck.FileExists(),
			},
			"the name of the file holding"+
				" the certificate for the certification authority (CA)",
			param.Attrs(param.MustBeSet))

		ps.Add(prefix+certInfoParamNameCertFilename,
			psetter.Pathname{
				Value:       &ci.certFilename,
				Expectation: filecheck.FileExists(),
			},
			"the name of the file holding"+
				" the certificate for the program",
			param.Attrs(param.MustBeSet))

		ps.Add(prefix+certInfoParamNameKeyFilename,
			psetter.Pathname{
				Value:       &ci.keyFilename,
				Expectation: filecheck.FileExists(),
			},
			"the name of the file holding"+
				" the private key for the program's certificate",
			param.Attrs(param.MustBeSet))

		return nil
	}
}
