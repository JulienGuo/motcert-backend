package main

import (
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"strconv"
)

type Certificate struct {
	CertId              string `protobuf:"bytes,1,req,name=certId" json:"certId"`                            //证书编号
	CertType            string `protobuf:"bytes,2,req,name=certType" json:"certType"`                        //证书类型
	EntrustOrg          string `protobuf:"bytes,3,opt,name=entrustOrg" json:"entrustOrg"`                    //委托单位
	InstrumentName      string `protobuf:"bytes,4,opt,name=instrumentName" json:"instrumentName"`            //器具名称
	Spec                string `protobuf:"bytes,5,opt,name=spec" json:"spec"`                                //型号/规格
	ExportId            string `protobuf:"bytes,6,opt,name=exportId" json:"exportId"`                        //出厂编号
	MadeByOrg           string `protobuf:"bytes,7,opt,name=madeByOrg" json:"madeByOrg"`                      //制造单位
	EntrustOrgAdd       string `protobuf:"bytes,8,opt,name=entrustOrgAdd" json:"entrustOrgAdd"`              //委托单位地址
	Approver            string `protobuf:"bytes,9,opt,name=approver" json:"approver"`                        //批准人
	Verifier            string `protobuf:"bytes,10,opt,name=verifier" json:"verifier"`                       //核验员
	CalibratePerson     string `protobuf:"bytes,11,opt,name=calibratePerson" json:"calibratePerson"`         //校准员
	CalibrateDate       string `protobuf:"bytes,12,opt,name=calibrateDate" json:"calibrateDate"`             //校准日期
	SuggestNextCaliDate string `protobuf:"bytes,13,opt,name=suggestNextCaliDate" json:"suggestNextCaliDate"` //建议下次校准日期
	IsCompleted         bool   `protobuf:"bytes,14,opt,name=isCompleted" json:"isCompleted"`                 //是否完成
	IsOpen              bool   `protobuf:"bytes,15,opt,name=isOpen" json:"isOpen"`                           //是否公开
	IsDeleted           bool   `protobuf:"bytes,16,opt,name=isDeleted" json:"isDeleted"`                     //是否删除
	UpdateDate          string `protobuf:"bytes,17,opt,name=updateDate" json:"updateDate"`                   //最新修改日期
}

type ListInternal struct {
	PageCount int
	Bookmark  string
	Certs     []Certificate
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

	switch function {
	case "postCertificate":
		return t.postCertificate(stub, args)
	case "getCertificate":
		return t.getCertificate(stub, args)
	case "queryList":
		return t.queryList(stub, args)
	default:
		return shim.Error("Unknown function: " + function)
	}
}

func (t *SimpleChaincode) postCertificate(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	fmt.Printf("postCertificate=%v\n", args)

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments.")
	}

	body := []byte(args[0])
	var certificate Certificate
	err := json.Unmarshal(body, &certificate)
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Printf("postCertificate: '%v'\n", certificate)

	attributes := []string{certificate.CertId}
	key, err := stub.CreateCompositeKey("certId", attributes)
	if err != nil {
		return shim.Error(err.Error())
	}

	certificateReqByte, err := json.Marshal(certificate)
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

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments.")
	}
	key, err := stub.CreateCompositeKey("certId", []string{args[0]})
	if err != nil {
		return shim.Error(err.Error())
	}

	value, err := stub.GetState(key)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(value)
}

func (t *SimpleChaincode) queryList(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	fmt.Printf("queryList=%v\n", args)
	fmt.Printf("queryList=%v\n", args[0])
	fmt.Printf("queryList=%v\n", args[1])
	fmt.Printf("queryList=%v\n", args[2])

	if len(args) < 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3")
	}

	queryString := args[0]

	pageSize, err := strconv.ParseInt(args[1], 10, 32)
	if err != nil {
		return shim.Error(err.Error())
	}
	bookmark := args[2]

	queryResults, err := getQueryResultForQueryStringWithPagination(stub, queryString, int32(pageSize), bookmark)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

// =========================================================================================
// getQueryResultForQueryStringWithPagination executes the passed in query string with
// pagination info. Result set is built and returned as a byte array containing the JSON results.
// =========================================================================================
func getQueryResultForQueryStringWithPagination(stub shim.ChaincodeStubInterface, queryString string, pageSize int32, bookmark string) ([]byte, error) {

	fmt.Printf("- getQueryResultForQueryString queryString:\n%s\n", queryString)

	resultsIterator, responseMetadata, err := stub.GetQueryResultWithPagination(queryString, pageSize, bookmark)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var Certs []Certificate
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		var cert Certificate
		err = json.Unmarshal(queryResponse.Value, &cert)
		if err != nil {
			return nil, err
		}
		Certs = append(Certs, cert)
	}
	list := ListInternal{
		int(responseMetadata.FetchedRecordsCount),
		responseMetadata.Bookmark,
		Certs,
	}

	bytesList, err := json.Marshal(list)

	return bytesList, err
}

func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}
