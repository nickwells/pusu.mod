package pusu

import (
	"testing"

	"github.com/nickwells/testhelper.mod/v2/testhelper"
)

func TestCertPool(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		testhelper.ExpErr
		testhelper.ExpPanic
		filename string
	}{
		{
			ID: testhelper.MkID("pool not populated"),
			ExpPanic: testhelper.MkExpPanic(
				"the certPool has not been successfully populated"),
		},
		{
			ID: testhelper.MkID("pool not populated - no such cert file"),
			ExpErr: testhelper.MkExpErr(
				`couldn't read the CA certificate file: "nonesuch":`,
				`open nonesuch: no such file or directory`),
			ExpPanic: testhelper.MkExpPanic(
				"the certPool has not been successfully populated"),
			filename: "nonesuch",
		},
		{
			ID: testhelper.MkID("pool not populated - bad cert file"),
			ExpErr: testhelper.MkExpErr(
				`the PEM is invalid: "testdata/badCertfile"`),
			ExpPanic: testhelper.MkExpPanic(
				"the certPool has not been successfully populated"),
			filename: "testdata/badCertfile",
		},
		{
			ID:       testhelper.MkID("pool populated"),
			filename: "testdata/goodCertfile",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ci := CertInfo{
				CACertFilename: tc.filename,
			}

			if tc.filename != "" {
				err := (&ci).PopulateCertPool()

				testhelper.CheckExpErr(t, err, tc)
			}

			panicked, panicVal := testhelper.PanicSafe(func() { ci.CertPool() })

			testhelper.CheckExpPanic(t, panicked, panicVal, tc)
		})
	}
}

func TestCert(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		testhelper.ExpErr
		testhelper.ExpPanic
		certFilename string
		keyFilename  string
		populate     bool
	}{
		{
			ID: testhelper.MkID("cert not populated"),
			ExpPanic: testhelper.MkExpPanic(
				"the cert has not been successfully populated"),
		},
		{
			ID: testhelper.MkID("no such cert file"),
			ExpPanic: testhelper.MkExpPanic(
				"the cert has not been successfully populated"),
			ExpErr: testhelper.MkExpErr(
				`couldn't load x509 keypair:`,
				`certFile: "nonesuch", keyFile: "",`,
				`err: open nonesuch: no such file or directory`),
			certFilename: "nonesuch",
		},
		{
			ID: testhelper.MkID("no such key file"),
			ExpPanic: testhelper.MkExpPanic(
				"the cert has not been successfully populated"),
			ExpErr: testhelper.MkExpErr(
				`couldn't load x509 keypair:`,
				`certFile: "testdata/badCertfile", keyFile: "nonesuch",`,
				`err: open nonesuch: no such file or directory`),
			certFilename: "testdata/badCertfile",
			keyFilename:  "nonesuch",
		},
		{
			ID: testhelper.MkID("bad key and cert file"),
			ExpPanic: testhelper.MkExpPanic(
				"the cert has not been successfully populated"),
			ExpErr: testhelper.MkExpErr(
				`couldn't load x509 keypair:`,
				`certFile: "testdata/badCertfile",`,
				`keyFile: "testdata/badKeyfile",`,
				`err: tls: failed to find any PEM data in certificate input`),
			certFilename: "testdata/badCertfile",
			keyFilename:  "testdata/badKeyfile",
		},
		// the following test fails (?) when the certificate expires.
		// To regenerate the certificate, run the following command
		// from within the testdata directory, it will generate a
		// certificate valid for one day.
		//
		// openssl req -x509 -new -nodes -key goodKeyfile \
		//             -sha256 -days 1 -out goodCertfile
		//
		// If the keyfile needs generating, that can be done by
		// running the following command from within the testdata
		// directory.
		//
		// openssl genrsa -out goodKeyfile 2048
		{
			ID:           testhelper.MkID("good key and cert file"),
			certFilename: "testdata/goodCertfile",
			keyFilename:  "testdata/goodKeyfile",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ci := CertInfo{
				CertFilename: tc.certFilename,
				KeyFilename:  tc.keyFilename,
			}

			if tc.certFilename != "" || tc.keyFilename != "" {
				err := (&ci).PopulateCert()

				testhelper.CheckExpErr(t, err, tc)
			}

			panicked, panicVal := testhelper.PanicSafe(func() { ci.Cert() })

			testhelper.CheckExpPanic(t, panicked, panicVal, tc)
		})
	}
}
