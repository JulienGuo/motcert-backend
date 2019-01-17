package main

import (
	"fmt"
	"gitlab.chainnova.com/motcert-backend/app/blockchain"
	"os"
	"encoding/hex"
	"io/ioutil"
	"paillier"
)

func main() {
	// Definition of the Fabric SDK properties
	fSetup := blockchain.FabricSetup{
		// Network parameters
		OrdererID: "orderer.cert.mot.gov.cn",

		// Channel parameters
		ChannelID:     "motcert",
		ChannelConfig: os.Getenv("GOPATH") + "/src/gitlab.chainnova.com/motcert-backend/fixtures/artifacts/motcert.channel.tx",

		// Chaincode parameters
		ChainCodeID:     "motcert-cc1",
		ChaincodeGoPath: os.Getenv("GOPATH"),
		ChaincodePath:   "gitlab.chainnova.com/motcert-backend/chaincode/",
		OrgAdmin:        "Admin",
		OrgName:         "org1",
		ConfigFile:      "../config.yaml",

		// User parameters
		UserName: "User1",
	}

	// Initialization of the Fabric SDK from the previously set properties
	err := fSetup.Initialize()
	if err != nil {
		fmt.Printf("Unable to initialize the Fabric SDK: %v\n", err)
		return
	}
	// Close SDK
	defer fSetup.CloseSDK()

	 //Install and instantiate the chaincode
	err = fSetup.InstallAndInstantiateCC()
	if err != nil {
		fmt.Printf("Unable to install and instantiate the chaincode: %v\n", err)
		return
	}

	fmt.Println("-----------------------------测试程序---------------------")
	fmt.Println("----------------------A用户开始注册密钥对信息，将密钥对信息注册上链---------------------------")
	//注册密钥对信息  A用户
	transactionID,err := fSetup.RegisterKeyPair("pubByteA","A")
	if err != nil {
		fmt.Printf("unable to invoke RegisterKeyPair1 %s \n",err)
	}else{
		fmt.Printf("Successful Invoke RegisterKeyPair1, transction ID :%s\n ",transactionID)
	}

	fmt.Println("-----------------------B用户开始注册密钥对信息，将密钥信息注册上链----------------------------")
	//注册密钥信息B用户
	transactionID,err = fSetup.RegisterKeyPair("pubByteB","B")
	if err != nil {
		fmt.Printf("unable to invoke RegisterKeyPair2 %s \n",err)
	}else{
		fmt.Printf("Successful Invoke RegisterKeyPair2, transction ID :%s\n ",transactionID)
	}

	fmt.Println("--------------------------------开始初始化用户参数--------------------------------------")
	transactionID,err = fSetup.InitParam("A",100,"B",200)

	if err != nil {
		fmt.Printf("unable to initParams %s \n",err)
	}else{
		fmt.Printf("Successful Invoke InitParams, transction ID :%s\n ",transactionID)
	}

	fmt.Println("--------------------------------查询用户数据----------------------")

	// Query again the chaincode
	response , err := fSetup.Query("B")

	queryReultByte,err := hex.DecodeString(response)

	privByte, err := ioutil.ReadFile("privateB.pem")
	if err != nil {
		fmt.Println(err)
	}

	queryResult , err := paillier.Decrypt(privByte,queryReultByte)
	if err != nil {
		fmt.Printf("Unable to query hello on the chaincode: %v\n", err)
	} else {
		fmt.Println("Response from the query B: ", queryResult)
	}


	fmt.Println("-------------------------------进行数据转账----------------------------")

	transactionID ,err = fSetup.Transfer("A","B","20","pubByteA","pubByteB")

	if err != nil {
		fmt.Printf("unable to transfer %s \n",err)
	}else{
		fmt.Printf("Successful Invoke transfer, transction ID :%s\n ",transactionID)
	}

	fmt.Println("--------------------------------查询用户数据----------------------")

	// Query again the chaincode
	response , err = fSetup.Query("A")

	queryReultByte,err = hex.DecodeString(response)

	privByteA, err := ioutil.ReadFile("privateA.pem")
	if err != nil {
		fmt.Println(err)
	}

	queryResult , err = paillier.Decrypt(privByteA,queryReultByte)
	if err != nil {
		fmt.Printf("Unable to query hello on the chaincode: %v\n", err)
	} else {
		fmt.Println("Response from the query A: ", queryResult)
	}

	fmt.Println("--------------------------------查询用户数据----------------------")

	// Query again the chaincode
	response , err = fSetup.Query("B")

	queryReultByte,err = hex.DecodeString(response)

	privByteB, err := ioutil.ReadFile("privateB.pem")
	if err != nil {
		fmt.Println(err)
	}

	queryResult , err = paillier.Decrypt(privByteB,queryReultByte)
	if err != nil {
		fmt.Printf("Unable to query hello on the chaincode: %v\n", err)
	} else {
		fmt.Println("Response from the query B: ", queryResult)
	}

}








































