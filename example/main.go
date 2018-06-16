package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/faceair/betproxy"
	"github.com/faceair/betproxy/mitm"
)

func main() {
	cacert, cakey, err := loadCA()
	if err != nil {
		panic(err)
	}
	tlsCfg, err := mitm.NewConfig(cacert, cakey)
	if err != nil {
		panic(err)
	}
	service, err := betproxy.NewService(":3128", tlsCfg)
	if err != nil {
		panic(err)
	}
	service.SetClient(&http.Client{})
	log.Fatal(service.Listen())
}

func generateCA() error {
	BetProxyCAPath := os.Getenv("HOME") + "/.betproxy"
	err := os.MkdirAll(BetProxyCAPath, os.ModePerm)
	if err != nil {
		return err
	}

	cacert, cakey, err := mitm.NewAuthority("betproxy", "faceair", 10*365*24*time.Hour)
	if err != nil {
		return err
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, cacert, cacert, &cakey.PublicKey, cakey)
	if err != nil {
		return err
	}
	certOut, err := os.Create(BetProxyCAPath + "/ca_cert.pem")
	if err != nil {
		return err
	}
	if err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return err
	}
	if err = certOut.Close(); err != nil {
		return err
	}

	keyOut, err := os.OpenFile(BetProxyCAPath+"/ca_key.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Print("failed to open key.pem for writing:", err)
	}
	if err = pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(cakey)}); err != nil {
		return err
	}

	return keyOut.Close()
}

func loadCA() (*x509.Certificate, *rsa.PrivateKey, error) {
	BetProxyCAPath := os.Getenv("HOME") + "/.betproxy"
	if _, err := os.Stat(BetProxyCAPath); os.IsNotExist(err) {
		err := generateCA()
		if err != nil {
			return nil, nil, err
		}
	}

	cert, err := ioutil.ReadFile(BetProxyCAPath + "/ca_cert.pem")
	if err != nil {
		return nil, nil, err
	}
	key, err := ioutil.ReadFile(BetProxyCAPath + "/ca_key.pem")
	if err != nil {
		return nil, nil, err
	}
	certBlock, _ := pem.Decode(cert)
	if certBlock == nil {
		return nil, nil, errors.New("Failed to decode ca certificate")
	}
	keyBlock, _ := pem.Decode(key)
	if keyBlock == nil {
		return nil, nil, errors.New("Failed to decode ca private key")
	}
	rawKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	rawCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}

	return rawCert, rawKey, nil
}
