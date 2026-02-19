package main

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns"
	"github.com/go-acme/lego/v4/registration"
	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
)

type MyUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (m *MyUser) GetEmail() string {
	return m.Email
}

func (m *MyUser) GetRegistration() *registration.Resource {
	return m.Registration
}

func (m *MyUser) GetPrivateKey() crypto.PrivateKey {
	return m.key
}

func main() {
	staging := flag.Bool("staging", false, "Set this to use staging")
	flag.Parse()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal(err)
	}

	myUser := MyUser{
		Email: "me@ramadhantriyant.id",
		key:   privateKey,
	}

	config := lego.NewConfig(&myUser)

	if *staging {
		config.CADirURL = lego.LEDirectoryStaging
	} else {
		config.CADirURL = lego.LEDirectoryProduction
	}

	config.Certificate.KeyType = certcrypto.RSA2048

	client, err := lego.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	// Fetch BUNNY_API_KEY from Vault
	vaultClient, err := vault.New(vault.WithEnvironment())
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	secret, err := vaultClient.Secrets.KvV2Read(ctx, "rsafe-ovh/dns/bunny", vault.WithMountPath("kv"))
	if err != nil {
		log.Fatal(err)
	}

	bunnyKey, ok := secret.Data.Data["BUNNY_API_KEY"].(string)
	if !ok {
		log.Fatal("BUNNY_API_KEY not found in Vault secret")
	}
	os.Setenv("BUNNY_API_KEY", bunnyKey)

	provider, err := dns.NewDNSChallengeProviderByName("bunny")
	if err != nil {
		log.Fatal(err)
	}

	client.Challenge.SetDNS01Provider(provider)

	// Register ACME account
	regOptions := registration.RegisterOptions{
		TermsOfServiceAgreed: true,
	}
	reg, err := client.Registration.Register(regOptions)
	if err != nil {
		log.Fatal(err)
	}
	myUser.Registration = reg

	request := certificate.ObtainRequest{
		Domains: []string{"*.rsafe.ovh"},
		Bundle:  true,
	}

	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Certificate obtained successfully for: %s\n", certificates.Domain)

	// Parse the certificate to extract metadata
	block, _ := pem.Decode(certificates.Certificate)
	if block == nil {
		log.Fatal("failed to decode certificate PEM")
	}
	x509Cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		log.Fatal(err)
	}
	serialHex := fmt.Sprintf("%X", x509Cert.SerialNumber)
	var serialFormatted strings.Builder
	for i, c := range serialHex {
		if i > 0 && i%2 == 0 {
			serialFormatted.WriteString(":")
		}
		serialFormatted.WriteRune(c)
	}

	// Store certificate data in Vault
	_, err = vaultClient.Secrets.KvV2Write(ctx, "rsafe-ovh/tls/rsafe.ovh",
		schema.KvV2WriteRequest{
			Data: map[string]any{
				"certificate": string(certificates.Certificate),
				"private_key": string(certificates.PrivateKey),
				"domains":     x509Cert.DNSNames,
				"issued_at":   x509Cert.NotBefore.UTC().Format("2006-01-02T15:04:05Z"),
				"expires_at":  x509Cert.NotAfter.UTC().Format("2006-01-02T15:04:05Z"),
				"serial":      serialFormatted.String(),
			},
		},
		vault.WithMountPath("kv"),
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Certificate stored in Vault at kv/rsafe-ovh/tls/rsafe.ovh (expires: %s)\n",
		x509Cert.NotAfter.UTC().Format("2006-01-02"))
}
