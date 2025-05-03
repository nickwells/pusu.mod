package pusu

import (
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/psetter"
)

const (
	clientConnInfoParamNameAddress = "pusu-server-address"
	paramNameTimeout               = "timeout"
)

// AddParams adds the parameters for the certificate info. The standard
// parameter names will be prefixed by the supplied prefix string (which
// should end with a dash); this is to allow multiple sets of parameters to
// be given
func (cci *ClientConnInfo) AddParams(prefix string) param.PSetOptFunc {
	return func(ps *param.PSet) error {
		ps.Add(prefix+clientConnInfoParamNameAddress,
			psetter.String[string]{
				Value: &cci.address,
			},
			"the address of the pub/sub server to connect to",
			param.Attrs(param.MustBeSet))

		ps.Add(paramNameTimeout,
			psetter.Duration{
				Value: &cci.timeout,
			},
			"the timeout. This is how long to wait before"+
				" abandoning the attempt to connect to"+
				" the pub/sub server",
			param.Attrs(param.DontShowInStdUsage))

		if err := cci.certInfo.AddParams("")(ps); err != nil {
			return err
		}

		return nil
	}
}
