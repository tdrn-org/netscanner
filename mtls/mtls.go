/*
 * Copyright 2025-2026 Holger de Carne
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mtls

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"math/big"
	"net"
	"os"
	"slices"
	"strings"
	"time"
)

type Credentials struct {
	CRTPEM      []byte
	Certificate *x509.Certificate
	KeyPEM      []byte
	Key         crypto.PrivateKey
	CAPEM       []byte
}

func LoadCredentials(crtFile, keyFile, caFile string) (*Credentials, error) {
	crtPEM, err := os.ReadFile(crtFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file '%s' (cause: %w)", crtFile, err)
	}
	certificate, err := decodeCertificate(crtPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to decode certificate read from file '%s' (cause: %w)", crtFile, err)
	}
	keyPEM, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file '%s' (cause: %w)", keyFile, err)
	}
	key, err := decodeKey(keyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to decode key read from file '%s' (cause: %w)", keyFile, err)
	}
	var caPEM []byte
	if caFile != "" {
		caPEM, err = os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate file '%s' (cause: %w)", caFile, err)
		}
	}
	credentials := &Credentials{
		CRTPEM:      crtPEM,
		Certificate: certificate,
		KeyPEM:      keyPEM,
		Key:         key,
		CAPEM:       caPEM,
	}
	return credentials, nil
}

func (c *Credentials) Write(crtFile, keyFile string, overwrite bool) error {
	createFlag := os.O_CREATE | os.O_WRONLY
	if overwrite {
		createFlag |= os.O_TRUNC
	} else {
		createFlag |= os.O_EXCL
	}
	crtIO, err := os.OpenFile(crtFile, createFlag, 0666)
	if err != nil {
		return fmt.Errorf("failed to create certificate file '%s' (cause: %w)", crtFile, err)
	}
	defer crtIO.Close()
	keyIO, err := os.OpenFile(keyFile, createFlag, 0600)
	if err != nil {
		return fmt.Errorf("failed to create key file '%s' (cause: %w)", keyFile, err)
	}
	_, err = crtIO.Write(c.CRTPEM)
	if err != nil {
		return fmt.Errorf("failed to write certificate file '%s' (cause: %w)", crtFile, err)
	}
	_, err = keyIO.Write(c.KeyPEM)
	if err != nil {
		return fmt.Errorf("failed to write key file '%s' (cause: %w)", crtFile, err)
	}
	return nil
}

func (c *Credentials) TLSConfig() (*tls.Config, error) {
	cert, err := tls.X509KeyPair(c.CRTPEM, c.KeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS key pair (cause: %w)", err)
	}
	var clientCAs *x509.CertPool
	if len(c.CAPEM) > 0 {
		clientCAs = x509.NewCertPool()
		if !clientCAs.AppendCertsFromPEM(c.CAPEM) {
			return nil, fmt.Errorf("failed to add TLS CA")
		}
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientCAs,
	}
	return tlsConfig, nil
}

type CommonOptions struct {
	ON       string
	CN       string
	IPs      []net.IP
	DNS      []string
	Validity time.Duration
}

func InitNodeOptions(on, cn string, validity time.Duration, ipFilter func(net.IP) bool) *CommonOptions {
	ips := listHostIPs(ipFilter)
	dns := listHostDNS(ips)
	return &CommonOptions{
		ON:       on,
		CN:       cn,
		IPs:      ips,
		DNS:      dns,
		Validity: validity,
	}
}

func listHostIPs(filter func(net.IP) bool) []net.IP {
	ips := make([]net.IP, 0)
	ifaces, err := net.Interfaces()
	if err != nil {
		slog.Warn("failed to list host interfaces", slog.Any("err", err))
		return ips
	}
	for _, iface := range ifaces {
		ips = append(ips, listHostInterfaceIPs(iface, filter)...)
	}
	slices.SortFunc(ips, func(ip1, ip2 net.IP) int { return strings.Compare(ip1.String(), ip2.String()) })
	return ips
}

func listHostInterfaceIPs(iface net.Interface, filter func(net.IP) bool) []net.IP {
	ips := make([]net.IP, 0)
	addrs, err := iface.Addrs()
	if err != nil {
		slog.Warn("failed to list interface addresses", slog.Any("err", err))
		return ips
	}
	for _, addr := range addrs {
		switch addr := addr.(type) {
		case *net.IPAddr:
			if filter(addr.IP) {
				ips = append(ips, addr.IP)
			}
		case *net.IPNet:
			if filter(addr.IP) {
				ips = append(ips, addr.IP)
			}
		}
	}
	slices.SortFunc(ips, func(ip1, ip2 net.IP) int { return strings.Compare(ip1.String(), ip2.String()) })
	return ips
}

func listHostDNS(ips []net.IP) []string {
	dnsMap := make(map[string]string, len(ips))
	for _, ip := range ips {
		names, _ := net.LookupAddr(ip.String())
		for _, name := range names {
			dnsName := strings.TrimSuffix(name, ".")
			dnsMap[dnsName] = name
		}
	}
	dns := slices.Collect(maps.Keys(dnsMap))
	slices.Sort(dns)
	return dns
}

func (o *CommonOptions) pkixName() pkix.Name {
	return pkix.Name{
		Organization: []string{o.ON},
		CommonName:   o.CN,
	}
}

type CAOptions struct {
	CommonOptions
}

func (o *CAOptions) Generate() (*Credentials, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create CA key pair (cause: %w)", err)
	}
	serial, err := generateSerial(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA serial (cause: %w)", err)
	}
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               o.pkixName(),
		IPAddresses:           o.IPs,
		DNSNames:              o.DNS,
		NotBefore:             now,
		NotAfter:              now.Add(o.Validity),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	crtBytes, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA certificate (cause: %w)", err)
	}
	certificate, err := x509.ParseCertificate(crtBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA certificate (cause: %w)", err)
	}
	crtPEM := pem.EncodeToMemory(&pem.Block{Type: certificateBlock, Bytes: crtBytes})
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal CA private key (cause: %w)", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: privateKeyBlock, Bytes: keyBytes})
	credentials := &Credentials{
		CRTPEM:      crtPEM,
		KeyPEM:      keyPEM,
		Certificate: certificate,
		Key:         key,
	}
	return credentials, nil
}

type NodeOptions struct {
	CommonOptions
	CA *Credentials
}

func (o *NodeOptions) Generate() (*Credentials, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create Node key pair (cause: %w)", err)
	}
	serial, err := generateSerial(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA serial (cause: %w)", err)
	}
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      o.pkixName(),
		IPAddresses:  o.IPs,
		DNSNames:     o.DNS,
		NotBefore:    now,
		NotAfter:     now.Add(o.Validity),
		IsCA:         true,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}
	caCrt, err := decodeCertificate(o.CA.CRTPEM)
	if err != nil {
		return nil, fmt.Errorf("failed access CA certificate (cause: %w)", err)
	}
	caKey, err := decodeKey(o.CA.KeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed access CA key (cause: %w)", err)
	}
	crtBytes, err := x509.CreateCertificate(rand.Reader, template, caCrt, &key.PublicKey, caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Node certificate (cause: %w)", err)
	}
	certificate, err := x509.ParseCertificate(crtBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Node certificate (cause: %w)", err)
	}
	crtPEM := pem.EncodeToMemory(&pem.Block{Type: certificateBlock, Bytes: crtBytes})
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Node private key (cause: %w)", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: privateKeyBlock, Bytes: keyBytes})
	credentials := &Credentials{
		CRTPEM:      crtPEM,
		KeyPEM:      keyPEM,
		Certificate: certificate,
		Key:         key,
	}
	return credentials, nil
}

const certificateBlock string = "CERTIFICATE"
const privateKeyBlock string = "EC PRIVATE KEY"

func generateSerial(random io.Reader) (*big.Int, error) {
	serial, err := rand.Int(random, big.NewInt(90000))
	if err != nil {
		return nil, err
	}
	serial.Add(serial, big.NewInt(10000))
	return serial, nil
}

func decodeCertificate(pemBytes []byte) (*x509.Certificate, error) {
	block, rest := pem.Decode(pemBytes)
	if block == nil || block.Type != certificateBlock || len(rest) != 0 {
		return nil, fmt.Errorf("invalid certificate")
	}
	return x509.ParseCertificate(block.Bytes)
}

func decodeKey(pemBytes []byte) (*ecdsa.PrivateKey, error) {
	block, rest := pem.Decode(pemBytes)
	if block == nil || block.Type != privateKeyBlock || len(rest) != 0 {
		return nil, fmt.Errorf("invalid private key")
	}
	return x509.ParseECPrivateKey(block.Bytes)
}
