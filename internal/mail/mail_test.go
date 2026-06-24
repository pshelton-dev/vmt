package mail

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"vmt/internal/config"
)

func selfSignedCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

// fakeSTARTTLS runs a minimal SMTP server that advertises and performs STARTTLS
// with the given (self-signed) cert. Received message bodies are sent to got.
func fakeSTARTTLS(t *testing.T, cert tls.Certificate, got chan<- string) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go serve(conn, cert, got)
		}
	}()
	return ln
}

func serve(conn net.Conn, cert tls.Certificate, got chan<- string) {
	defer conn.Close()
	br := bufio.NewReader(conn)
	write := func(s string) { conn.Write([]byte(s)) }
	readLine := func() string { l, _ := br.ReadString('\n'); return strings.TrimRight(l, "\r\n") }

	write("220 test ESMTP\r\n")
	for {
		up := strings.ToUpper(readLine())
		switch {
		case strings.HasPrefix(up, "EHLO"), strings.HasPrefix(up, "HELO"):
			write("250-test\r\n250 STARTTLS\r\n")
		case up == "STARTTLS":
			write("220 Go ahead\r\n")
			tc := tls.Server(conn, &tls.Config{Certificates: []tls.Certificate{cert}})
			if err := tc.Handshake(); err != nil {
				return // client rejected our self-signed cert
			}
			conn = tc
			br = bufio.NewReader(conn)
			write = func(s string) { tc.Write([]byte(s)) }
			readLine = func() string { l, _ := br.ReadString('\n'); return strings.TrimRight(l, "\r\n") }
		case strings.HasPrefix(up, "MAIL"), strings.HasPrefix(up, "RCPT"):
			write("250 OK\r\n")
		case up == "DATA":
			write("354 end with .\r\n")
			var sb strings.Builder
			for {
				dl := readLine()
				if dl == "." {
					break
				}
				sb.WriteString(dl + "\n")
			}
			write("250 OK\r\n")
			got <- sb.String()
		case up == "QUIT":
			write("221 Bye\r\n")
			return
		default:
			write("250 OK\r\n")
		}
	}
}

func TestSendSTARTTLSSelfSigned(t *testing.T) {
	cert := selfSignedCert(t)
	got := make(chan string, 1)
	ln := fakeSTARTTLS(t, cert, got)
	defer ln.Close()
	host, port, _ := net.SplitHostPort(ln.Addr().String())

	base := config.SMTP{Host: host, Port: port, From: "vmt@test", TLS: "starttls"}

	// 1) Verification ON (default) must fail against the self-signed cert.
	if err := Send(base, "to@test", "Subj", "Body"); err == nil {
		t.Fatal("expected a TLS verification error without VMT_SMTP_INSECURE, got nil")
	}

	// 2) Insecure ON must succeed and deliver the message.
	insecure := base
	insecure.Insecure = true
	if err := Send(insecure, "to@test", "Subj", "Hello body"); err != nil {
		t.Fatalf("expected success with Insecure=true, got: %v", err)
	}
	select {
	case msg := <-got:
		if !strings.Contains(msg, "Subject: Subj") || !strings.Contains(msg, "Hello body") {
			t.Fatalf("delivered message missing expected content:\n%s", msg)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for delivered message")
	}
}
