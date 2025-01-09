/*
Copyright 2025 The KCP Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package certificates

import (
	"crypto/x509"
	"time"

	"github.com/kcp-dev/api-syncagent/internal/certificates/triple"

	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	minimumCertValidity = 30 * 24 * time.Hour
)

type CAGetter func() (*triple.KeyPair, error)
type CACertGetter func() (*x509.Certificate, error)

// IsClientCertificateValidForAllOf validates if the given data matches exactly the given client certificate
// (It also returns true if all given data is in the cert, but the cert has more organizations).
func IsClientCertificateValidForAllOf(cert *x509.Certificate, commonName string, organizations []string, ca *x509.Certificate) (bool, error) {
	if certWillExpireSoon(cert) {
		return false, nil
	}

	if cert.Subject.CommonName != commonName {
		return false, nil
	}

	wantOrganizations := sets.New(organizations...)
	certOrganizations := sets.New(cert.Subject.Organization...)

	if !wantOrganizations.Equal(certOrganizations) {
		return false, nil
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(ca)
	verifyOptions := x509.VerifyOptions{
		Roots:     certPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	if _, err := cert.Verify(verifyOptions); err != nil {
		return false, err
	}

	return true, nil
}

// certWillExpireSoon returns if the certificate will expire in the next 30 days.
func certWillExpireSoon(cert *x509.Certificate) bool {
	return time.Until(cert.NotAfter) < minimumCertValidity
}
