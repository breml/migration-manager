package util

import (
	"crypto/x509"
	"log/slog"
	"time"

	incustls "github.com/lxc/incus/v6/shared/tls"
)

// CheckTrustState checks whether the given client certificate is trusted
// (i.e. it has a valid time span and it belongs to the given list of trusted
// certificates).
// Returns whether or not the certificate is trusted, and the fingerprint of the certificate.
func CheckTrustState(cert x509.Certificate, trustedCertFingerprints []string, ca *x509.Certificate, trustCACertificates bool) (bool, string) {
	// Extra validity check (should have been caught by TLS stack)
	if time.Now().Before(cert.NotBefore) || time.Now().After(cert.NotAfter) {
		return false, ""
	}

	certFingerprint := incustls.CertFingerprint(&cert)

	if ca != nil && trustCACertificates {
		// TODO: implement verification against CA and CRL following the logic in
		// https://github.com/lxc/incus/blob/e35b3540c052049aaa7202272ba693be9d0f8383/shared/tls/cert.go#L50
		panic("verification against CA and CRL is not yet implemented")
	}

	// Check whether client certificate fingerprint is trusted.
	for _, fingerprint := range trustedCertFingerprints {
		if certFingerprint == fingerprint {
			slog.Debug("Matched trusted cert", slog.String("fingerprint", fingerprint), slog.Any("subject", cert.Subject))
			return true, fingerprint
		}
	}

	return false, ""
}
