package paillier

import (
	"crypto/rand"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"testing"
)

func TestVerify(t *testing.T) {

}

type sm2Signature struct {
	R, S *big.Int
}

func (pub *PublicKey) Verify(msg []byte, sign []byte) bool {
	var sm2Sign sm2Signature
	_, err := asn1.Unmarshal(sign, &sm2Sign)
	if err != nil {
		return false
	}
	return true
}

func TestMarshalPrivateKey(t *testing.T) {
	privKey, _ := GenerateKey(rand.Reader, 2048)
	res := MarshalPrivateKey(privKey)

	block := &pem.Block{
		Type:  "PrivateKey",
		Bytes: res,
	}
	file, err := os.Create("PrivateKey1.pem")
	if err != nil {
		fmt.Println(err)
	}
	err = pem.Encode(file, block)
	if err != nil {
		fmt.Println(err)
	}

}

func TestGenPemPrivateKey(t *testing.T) {
	privKey, _ := GenerateKey(rand.Reader, 2048)
	res := GenPemPrivateKey(privKey)
	file1, err := os.Create("PrivateKey2.pem")
	if err != nil {
		fmt.Println(err)
	}
	defer file1.Close()
	file1.Write(res)

	pub := GenPemPublicKey(&privKey.PublicKey)
	file2, err := os.Create("PublicKey2.pem")
	if err != nil {
		fmt.Println(err)
	}
	defer file2.Close()
	file2.Write(pub)

}

func TestParseKey(t *testing.T) {
	privByte, err := ioutil.ReadFile("PrivateKey2.pem")
	if err != nil {
		fmt.Println(err)
	}
	_, err = ParsePrivateKey(privByte)
	if err != nil {
		fmt.Println(err)
	}

	pubByte, err := ioutil.ReadFile("PublicKey2.pem")
	if err != nil {
		fmt.Println(err)
	}

	_, err = ParsePublicKey(pubByte)
	if err != nil {
		fmt.Println(err)
	}
	cipher, err := Encrypt(pubByte, []byte("ChainNova"))
	if err != nil {
		fmt.Println(err)
	}
	res, _ := Decrypt(privByte, cipher)
	fmt.Println("decode string", string(res))
}
