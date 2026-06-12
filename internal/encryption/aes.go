package encryption

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "errors"
    "io"
)

type AESService struct {
    key []byte
}

func NewAESService(key string) *AESService {
    keyBytes := []byte(key)
    if len(keyBytes) < 32 {
        padded := make([]byte, 32)
        copy(padded, keyBytes)
        keyBytes = padded
    }
    return &AESService{key: keyBytes[:32]}
}

func (s *AESService) Encrypt(plaintext string) (string, []byte, error) {
    block, err := aes.NewCipher(s.key)
    if err != nil {
        return "", nil, err
    }
    
    iv := make([]byte, aes.BlockSize)
    if _, err := io.ReadFull(rand.Reader, iv); err != nil {
        return "", nil, err
    }
    
    padded := pkcs7Pad([]byte(plaintext), aes.BlockSize)
    ciphertext := make([]byte, len(padded))
    mode := cipher.NewCBCEncrypter(block, iv)
    mode.CryptBlocks(ciphertext, padded)
    
    return base64.StdEncoding.EncodeToString(ciphertext), iv, nil
}

func (s *AESService) Decrypt(ciphertext string, iv []byte) (string, error) {
    block, err := aes.NewCipher(s.key)
    if err != nil {
        return "", err
    }
    
    decoded, err := base64.StdEncoding.DecodeString(ciphertext)
    if err != nil {
        return "", err
    }
    
    if len(iv) != aes.BlockSize {
        return "", errors.New("invalid IV length")
    }
    
    plaintext := make([]byte, len(decoded))
    mode := cipher.NewCBCDecrypter(block, iv)
    mode.CryptBlocks(plaintext, decoded)
    
    unpadded, err := pkcs7Unpad(plaintext)
    if err != nil {
        return "", err
    }
    
    return string(unpadded), nil
}

func pkcs7Pad(data []byte, blockSize int) []byte {
    padding := blockSize - len(data)%blockSize
    padtext := make([]byte, len(data)+padding)
    copy(padtext, data)
    for i := len(data); i < len(padtext); i++ {
        padtext[i] = byte(padding)
    }
    return padtext
}

func pkcs7Unpad(data []byte) ([]byte, error) {
    length := len(data)
    if length == 0 {
        return nil, errors.New("invalid padding")
    }
    
    padding := int(data[length-1])
    if padding > length {
        return nil, errors.New("invalid padding")
    }
    
    return data[:length-padding], nil
}
