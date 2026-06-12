package encryption

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
)

type RSAService struct {
	keySize int
}

func NewRSAService(keySize int) *RSAService {
	return &RSAService{keySize: keySize}
}

func (s *RSAService) GenerateKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, s.keySize)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, &privateKey.PublicKey, nil
}

func (s *RSAService) ExportPublicKey(pubKey *rsa.PublicKey) (string, error) {
	pubASN1, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return "", err
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubASN1,
	})
	return string(pubPEM), nil
}

func (s *RSAService) ExportPrivateKey(privKey *rsa.PrivateKey) (string, error) {
	privASN1 := x509.MarshalPKCS1PrivateKey(privKey)
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privASN1,
	})
	return string(privPEM), nil
}

func (s *RSAService) ImportPublicKey(pubKeyStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pubKeyStr))
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pubKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not RSA public key")
	}
	return pubKey, nil
}

func (s *RSAService) ImportPrivateKey(privKeyStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privKeyStr))
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return privKey, nil
}

func (s *RSAService) EncryptWithPublicKey(plaintext string, pubKey *rsa.PublicKey) (string, error) {
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, []byte(plaintext), nil)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *RSAService) DecryptWithPrivateKey(ciphertext string, privKey *rsa.PrivateKey) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privKey, decoded, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
