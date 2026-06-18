package plugins

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
)

// 加密base64字符串
func EncodeStr2Base64(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}

// 解密base64字符串
func DecodeStrFromBase64(str string) string {
	decodeBytes, _ := base64.StdEncoding.DecodeString(str)
	return string(decodeBytes)
}

// RSA加密
func RSAEncrypt(data, publicBytes []byte) ([]byte, error) {
	var res []byte
	// 解析公钥
	block, _ := pem.Decode(publicBytes)
	if block == nil {
		return res, errors.New("无法加密, 公钥可能不正确")
	}

	keyInit, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return res, fmt.Errorf("无法加密, 公钥可能不正确, %v", err)
	}

	//使用公钥加密数据
	pubKey := keyInit.(*rsa.PublicKey)
	res, err = rsa.EncryptPKCS1v15(rand.Reader, pubKey, data)
	if err != nil {
		return res, fmt.Errorf("无法加密, 公钥可能不正确, %v", err)
	}
	// 将数据加密为base64
	return []byte(EncodeStr2Base64(string(res))), nil
}

// RSA解密
func RSADecrypt(bse64dData, privateBytes []byte) ([]byte, error) {
	var res []byte
	// 解析base64
	data := []byte(DecodeStrFromBase64(string(bse64dData)))

	// 解析私钥
	block, _ := pem.Decode(privateBytes)
	if block == nil {
		return res, fmt.Errorf("无法解密, 私钥可能不正确,解析私钥失败")
	}

	// 解密数据
	keyInit, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return res, fmt.Errorf("无法解密, 私钥可能不正确, %v", err)
	}
	res, err = rsa.DecryptPKCS1v15(rand.Reader, keyInit, data)
	if err != nil {
		return res, fmt.Errorf("无法解密, 私钥可能不正确,解析PKCS失败 %v", err)
	}
	return res, nil
}

// 验签
func VerifierSign(dataDecrypted string, publicBytes []byte, sign string) (bool, error) {
	signBytes, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return false, fmt.Errorf("签名解码失败: %w\n", err)
	}

	// 解析公钥
	block, _ := pem.Decode(publicBytes)
	if block == nil {
		return false, errors.New("公钥解析失败")
	}
	// 解析公钥内容
	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return false, fmt.Errorf("公钥内容解析失败: %w\n", err)
	}
	// 类型断言为RSA公钥
	rsaPubKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return false, fmt.Errorf("不是有效的RSA公钥\n")
	}
	dataHash := sha256.Sum256([]byte(dataDecrypted))
	dataHashBytes := dataHash[:]
	// RSA公钥，算法，原始数据哈希值，待验证的签名
	err = rsa.VerifyPKCS1v15(rsaPubKey, crypto.SHA256, dataHashBytes, signBytes)
	if err != nil {
		return false, fmt.Errorf("验签错误: %w\n", err)
	}
	return true, nil
}
