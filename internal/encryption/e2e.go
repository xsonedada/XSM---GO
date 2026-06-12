package encryption

import (
    "crypto"
    "crypto/rand"
    "crypto/rsa"
    "crypto/sha256"
    "encoding/base64"
)

type E2EService struct {
    aes *AESService
    rsa *RSAService
}

func NewE2EService(key string, rsaKeySize int) *E2EService {
    return &E2EService{
        aes: NewAESService(key),
        rsa: NewRSAService(rsaKeySize),
    }
}

func (s *E2EService) GenerateMessageKey() (string, error) {
    key := make([]byte, 32)
    if _, err := rand.Read(key); err != nil {
        return "", err
    }
    return base64.StdEncoding.EncodeToString(key), nil
}

func (s *E2EService) EncryptMessage(plaintext string, messageKey string) (string, []byte, error) {
    tempAES := NewAESService(messageKey)
    return tempAES.Encrypt(plaintext)
}

func (s *E2EService) DecryptMessage(ciphertext string, iv []byte, messageKey string) (string, error) {
    tempAES := NewAESService(messageKey)
    return tempAES.Decrypt(ciphertext, iv)
}

func (s *E2EService) GenerateSignature(data string, privateKey string) (string, error) {
    privKey, err := s.rsa.ImportPrivateKey(privateKey)
    if err != nil {
        return "", err
    }
    hash := sha256.Sum256([]byte(data))
    signature, err := rsa.SignPKCS1v15(rand.Reader, privKey, crypto.SHA256, hash[:])
    if err != nil {
        return "", err
    }
    return base64.StdEncoding.EncodeToString(signature), nil
}

func (s *E2EService) VerifySignature(data, signature string, publicKey string) error {
    pubKey, err := s.rsa.ImportPublicKey(publicKey)
    if err != nil {
        return err
    }
    sigBytes, err := base64.StdEncoding.DecodeString(signature)
    if err != nil {
        return err
    }
    hash := sha256.Sum256([]byte(data))
    return rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hash[:], sigBytes)
}
