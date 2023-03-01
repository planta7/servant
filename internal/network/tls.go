// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package network

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"os"
	"time"
)

const (
	TemplateCountry      = "ES"
	TemplateOrg          = "planta7"
	TemplateOrgUnit      = "serve"
	CertificateBlockType = "CERTIFICATE"
	KeyBlockType         = "EC PRIVATE KEY"
	TempFilePattern      = "auto-tls-*"
)

func GenerateAutoTLS() (string, string) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal("Something went wrong while generating a key", "err", err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Country:            []string{TemplateCountry},
			Organization:       []string{TemplateOrg},
			OrganizationalUnit: []string{TemplateOrgUnit},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 180),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certificate := generateCertificate(&template, privateKey)
	certFile := writeToFile(certificate)

	key := generateKey(privateKey)
	keyFile := writeToFile(key)

	return certFile.Name(), keyFile.Name()
}

func generateCertificate(template *x509.Certificate, privateKey *ecdsa.PrivateKey) *bytes.Buffer {
	derBytes, err := x509.CreateCertificate(
		rand.Reader,
		template,
		template,
		&privateKey.PublicKey,
		privateKey)
	if err != nil {
		log.Fatal("Something went wrong while generating a TLS certificate", "err", err)
	}
	certificate := &bytes.Buffer{}
	_ = pem.Encode(certificate, &pem.Block{Type: CertificateBlockType, Bytes: derBytes})
	return certificate
}

func generateKey(privateKey *ecdsa.PrivateKey) *bytes.Buffer {
	b, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		log.Fatal("Unable to marshal ECDSA private key", "err", err)
	}
	pemBlockForKey := &pem.Block{Type: KeyBlockType, Bytes: b}
	key := &bytes.Buffer{}
	_ = pem.Encode(key, pemBlockForKey)
	return key
}

func writeToFile(content *bytes.Buffer) *os.File {
	file, err := os.CreateTemp("", TempFilePattern)
	if err != nil {
		log.Fatal(err)
	}
	_, err = file.Write(content.Bytes())
	_ = file.Close()
	return file
}
