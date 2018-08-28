// +build windows

package http

import (
	"crypto/x509"
	"syscall"
	"unsafe"
)

func loadSystemRoots() (*x509.CertPool, error) {
	const CRYPT_E_NOT_FOUND = 0x80092004
	store, err := syscall.CertOpenSystemStore(0, syscall.StringToUTF16Ptr("ROOT"))
	if err != nil {
		return nil, err
	}
	defer syscall.CertCloseStore(store, 0)
	roots := x509.NewCertPool()
	var cert *syscall.CertContext
	for {
		cert, err = syscall.CertEnumCertificatesInStore(store, cert)
		if err != nil {
			if errno, ok := err.(syscall.Errno); ok {
				if errno == CRYPT_E_NOT_FOUND {
					break
				}
			}
			return nil, err
		}
		if cert == nil {
			break
		}
		// Copy the buf, since ParseCertificate does not create its own copy.
		buf := (*[1 << 20]byte)(unsafe.Pointer(cert.EncodedCert))[:]
		buf2 := make([]byte, cert.Length)
		copy(buf2, buf)
		if c, err := x509.ParseCertificate(buf2); err == nil {
			roots.AddCert(c)
		}
	}
	return roots, nil
}
