package blockchain

import (
	"os"
	"crypto/rand"
	"paillier"
	"io/ioutil"
	"fmt"
	"encoding/hex"
	"time"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
)

//注册密钥对，将公钥信息写入到链上
func (setup *FabricSetup)RegisterKeyPair(pubByteKeyParam string,name string)(string,error){

	fmt.Println("-----------准备生成密钥对---------------")
	err := GenKey(64,name)
	if err != nil {
		return  "",err
	}
	//privByte, err := ioutil.ReadFile("private"+name+".pem")
	//if err != nil {
	//	fmt.Println(err)
	//}
	//privByteStr := hex.EncodeToString(privByte)

	pubByte ,err := ioutil.ReadFile("public"+name+".pem")
	if err !=nil {
		fmt.Println(err.Error())
	}

	pubByteStr := hex.EncodeToString(pubByte)

	fmt.Println("--------------生成密钥对成功---------------")

	eventID := "RegisterKeyPair"

	fmt.Println("------------开始准备执行参数------------------")
	//Prepare arguments
	var args []string
	args = append(args,"invoke")
	args = append(args,"registerKeyPair")
	args = append(args,pubByteKeyParam)
	args = append(args,pubByteStr)
	//args = append(args,priByteKeyParam)
	//args = append(args,privByteStr)

	fmt.Println("------------准备执行参数完成：------------------")

	// Add data that will be visible in the proposal, like a description of the invoke request
	transientDataMap := make(map[string][]byte)
	transientDataMap["result"] = []byte("Transient data in registerKeyPair")


	reg , notifier ,err :=setup.event.RegisterChaincodeEvent(setup.ChainCodeID,eventID)

	if err != nil {
		return "", err
	}
	defer setup.event.Unregister(reg)


	// Create a request (proposal) and send it
	response, err := setup.client.Execute(channel.Request{ChaincodeID: setup.ChainCodeID, Fcn: args[0], Args: [][]byte{[]byte(args[1]), []byte(args[2]), []byte(args[3])}, TransientMap: transientDataMap})
	if err != nil {
		return "", fmt.Errorf("failed to move funds  (CREATE KEYPAIR): %v", err)
	}

	// Wait for the result of the submission
	select {
	case ccEvent := <-notifier:

			fmt.Printf("Received CC event: %s\n", ccEvent)

		case <-time.After(time.Second * 20):

			return  "",fmt.Errorf("did NOT receive CC event for eventId(%s)", eventID)
	}


	return string(response.TransactionID),nil
}


func GenKey(bits int,name string) error {
	// 生成私钥文件
	privateKey, err := paillier.GenerateKey(rand.Reader, bits)
	if err != nil {
		return err
	}
	derStream := paillier.GenPemPrivateKey(privateKey)
	file, err := os.Create("private"+name+".pem")
	defer file.Close()
	if err != nil {
		return err
	}
	file.Write(derStream)

	defer file.Close()

	// 生成公钥文件
	publicKey := &privateKey.PublicKey

	derPkix := paillier.GenPemPublicKey(publicKey)
	file, err = os.Create("public"+name+".pem")
	if err != nil {
		return err
	}

	file.Write(derPkix)

	return nil
}

