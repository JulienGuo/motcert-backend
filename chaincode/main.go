package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"math/big"
	"strconv"
)

type CertificateReq struct {
	Id string `protobuf:"bytes,1,opt,name=id" json:"id"`
}

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

//instantiate chaincode
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("###############Chaincode Init###################")
	// Get the function and arguments from the request
	function, _ := stub.GetFunctionAndParameters()

	if function != "init" {
		return shim.Error("Unknown function call")
	}

	fmt.Println("###############Chaincode Init Success##################")
	return shim.Success(nil)
}

// invoke chaincode
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {

	fmt.Println("###########  Invoke ###########")
	function, args := stub.GetFunctionAndParameters()
	// Get the function and arguments from the request
	if function != "invoke" {
		return shim.Error("Unknown function call")
	}

	// Check whether the number of arguments is sufficient
	if len(args) < 1 {
		return shim.Error("The number of arguments is insufficient.")
	}

	switch args[0] {
	case "registerKeyPair":
		return t.registerKeyPair(stub, args)
	case "initParam":
		return t.initParam(stub, args)
	case "query":
		return t.query(stub, args)
	case "transfer":
		return t.transfer(stub, args)
	case "postCertificate":
		return t.postCertificate(stub, args)
	case "getCertificate":
		return t.getCertificate(stub, args)
	default:
		return shim.Error("Unknown function: " + function)
	}
}

//注册密钥对信息，将公钥对信息注册到链上  4个参数
func (t *SimpleChaincode) registerKeyPair(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	fmt.Println("########################开始注册密钥对######################")

	var pubByteKeyStr, pubByteValueStr string

	pubByteKeyStr = args[1]
	pubByteValueStr = args[2]

	//将公钥信息转码
	pubByteValue, _ := hex.DecodeString(pubByteValueStr)

	//将公钥信息注册链上
	err := stub.PutState(pubByteKeyStr, pubByteValue)
	if err != nil {
		shim.Error(err.Error())
	}

	//var priByteKeyStr , priByteValueStr string
	//
	//priByteKeyStr = args[3]
	//priByteValueStr = args[4]
	//
	////将私钥信息进行转码----------测试数据
	//priByteValue ,_ := hex.DecodeString(priByteValueStr)
	//
	////测试将私钥信息注册----------测试数据
	//err = stub.PutState(priByteKeyStr,priByteValue)
	//if err != nil {
	//	shim.Error(err.Error())
	//}

	err = stub.SetEvent("RegisterKeyPair", []byte{})
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(pubByteValue)
}

//init 进行参数实例化操作,在sdk层对数据进行加密操作写入链上
func (t *SimpleChaincode) initParam(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	fmt.Println("#############开始参数实例化###############")

	//// Get the function and arguments from the request
	//function, args := stub.GetFunctionAndParameters()
	//
	//// Check if the request is the init function
	//if function != "initParam" {
	//	return shim.Error("Unknown function call")
	//}
	//
	//if len(args) != 4 {
	//	return shim.Error("Incorrect number of arguments. Expecting 4")
	//}

	var A, B string
	var ciperAvalStr, ciperBvalStr string
	A = args[1]
	ciperAvalStr = args[2]

	ciperAval, err := hex.DecodeString(ciperAvalStr)
	if err != nil {
		return shim.Error(err.Error())
	}

	B = args[3]
	ciperBvalStr = args[4]
	ciperBval, err := hex.DecodeString(ciperBvalStr)
	if err != nil {
		return shim.Error(err.Error())
	}

	err = stub.PutState(A, ciperAval)
	if err != nil {
		return shim.Error(err.Error())
	}

	err = stub.PutState(B, ciperBval)
	if err != nil {
		return shim.Error(err.Error())
	}

	err = stub.SetEvent("initParamEvent", []byte{})
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success([]byte("#####################参数实例化成功##################"))

}

// query callback representing the query of a chaincode
func (t *SimpleChaincode) query(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var A string // Entities
	var err error

	//获取用户名
	A = args[1]

	//测试  获取私钥
	//priByteKeyStr := "priByte"+args[1]
	//
	//fmt.Println("------私钥用户------",priByteKeyStr)
	//
	//priByteKey ,_ := stub.GetState(priByteKeyStr)

	// Get the state from the ledger
	ciperAvalbytes, err := stub.GetState(A)
	if err != nil {
		jsonResp := "{\"Error\":\"Failed to get state for " + A + "\"}"
		return shim.Error(jsonResp)
	}

	if ciperAvalbytes == nil {
		jsonResp := "{\"Error\":\"Nil amount for " + A + "\"}"
		return shim.Error(jsonResp)
	}

	jsonResp := "{\"Name\":\"" + A + "\",\"Amount\":\"" + new(big.Int).SetBytes(ciperAvalbytes).String() + "\"}"

	fmt.Printf("Query Response:%s\n", jsonResp)
	fmt.Println("-----查询后的加密结果-------", new(big.Int).SetBytes(ciperAvalbytes).String())

	//进行测试  通过私钥解密数据
	//plainAval ,_ := stub.Decrypt(priByteKey,ciperAvalbytes)
	//
	//fmt.Println("------解密出来的结果是------",plainAval)
	//
	transResult := hex.EncodeToString(ciperAvalbytes)

	return shim.Success([]byte(transResult))

}

// 将转账金额传入，并且随机数传入r1,r2   chaincode端进行接受 A , B , X , r1 ,r2 ,pubByteAKey ,pubByteBKey
func (t *SimpleChaincode) transfer(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	var A, B string  // Entities
	var X int64      // Transaction value
	var r1, r2 int64 //random number
	//var pubByteAStr ,pubByteBStr string
	var err error

	A = args[1]
	B = args[2]

	// Get the state from the ledger
	// TODO: will be nice to have a GetAllState call to ledger

	//查询出来的是同态加密的结果
	ciperAvalbytes, err := stub.GetState(A)
	if err != nil {
		return shim.Error("Failed to get state")
	}
	if ciperAvalbytes == nil {
		return shim.Error("Entity not found")
	}

	ciperBvalbytes, err := stub.GetState(B)
	if err != nil {
		return shim.Error("Failed to get state")
	}
	if ciperBvalbytes == nil {
		return shim.Error("Entity not found")
	}

	// Perform the execution
	intX, err := strconv.Atoi(args[3])
	if err != nil {
		return shim.Error("Invalid transaction amount, expecting a integer value")
	}
	X = int64(intX)
	plianX := new(big.Int).SetInt64(X)

	r1Str := args[4]
	r2Str := args[5]
	r1Int, err := strconv.Atoi(r1Str)
	if err != nil {
		return shim.Error(err.Error())
	}
	r2Int, err := strconv.Atoi(r2Str)
	if err != nil {
		return shim.Error(err.Error())
	}

	r1 = int64(r1Int)
	r2 = int64(r2Int)

	r1B := new(big.Int).SetInt64(r1)
	r2B := new(big.Int).SetInt64(r2)

	pubByteAKey := args[6]
	pubByteA, err := stub.GetState(pubByteAKey)
	if err != nil {
		return shim.Error(err.Error())
	}

	pubByteBKey := args[7]
	pubByteB, err := stub.GetState(pubByteBKey)

	if err != nil {
		return shim.Error(err.Error())
	}

	//同态加密随机值数据处理
	ciperX1, _ := stub.Encrypt(pubByteA, plianX.Bytes(), r1B)
	fmt.Printf("ciperX1 = %s", new(big.Int).SetBytes(ciperX1).String())

	ciperX2, _ := stub.Encrypt(pubByteB, plianX.Bytes(), r2B)
	fmt.Printf("ciperX2 = %s", new(big.Int).SetBytes(ciperX2).String())

	fmt.Printf("ciperAvalbytes = %s ,ciperBvalbytes = %s", new(big.Int).SetBytes(ciperAvalbytes).String(), new(big.Int).SetBytes(ciperBvalbytes).String())

	//Aval = Aval - X
	//Bval = Bval + X

	fmt.Printf("----------------start transfer-------------------------------")

	//A-X
	ciperAvalbytes, _ = stub.SubCipher(pubByteA, ciperAvalbytes, ciperX1)
	//// B+X
	ciperBvalbytes, _ = stub.AddCipher(pubByteB, ciperBvalbytes, ciperX2)

	// Write the state back to the ledger
	//将加密数据存储到账本当中
	err = stub.PutState(A, ciperAvalbytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	err = stub.PutState(B, ciperBvalbytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	err = stub.SetEvent("transferEvent", []byte{})
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success([]byte("invoke transfer success"))
}

func (t *SimpleChaincode) postCertificate(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	fmt.Printf("postCertificate=%v\n", args)

	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments.")
	}

	// TODO: check duplication
	//txID := stub.GetTxID()

	body := []byte(args[1])
	var certificateReq CertificateReq
	err := json.Unmarshal(body, &certificateReq)
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Printf("postCertificate: '%v'\n", certificateReq)

	attributes := []string{certificateReq.Id}
	key, err := stub.CreateCompositeKey("certificate", attributes)
	if err != nil {
		return shim.Error(err.Error())
	}

	certificateReqByte, err := json.Marshal(certificateReq)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.PutState(key, certificateReqByte)
	if err != nil {
		return shim.Error(err.Error())
	}

	err = stub.SetEvent("postCertificateEvent", []byte{})
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(nil)
}

func (t *SimpleChaincode) getCertificate(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	fmt.Printf("getCertificate=%v\n", args)

	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments.")
	}

	key, err := stub.CreateCompositeKey("certificate", []string{args[1]})
	if err != nil {
		return shim.Error(err.Error())
	}

	value, err := stub.GetState(key)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(value)
}

func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}
