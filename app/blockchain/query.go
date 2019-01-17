package blockchain

import (
	"fmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
)

func (setup *FabricSetup)Query(username string)(string,error){

	//Prepare arguments
	var args []string
	args = append(args,"invoke")
	args = append(args,"query")
	args = append(args,username)

	response , err :=setup.client.Query(channel.Request{ChaincodeID:setup.ChainCodeID,Fcn:args[0],Args:[][]byte{[]byte(args[1]),[]byte(args[2])}})
	if err != nil {
		return "" ,fmt.Errorf("failed to query :%v",err)
	}
	return string(response.Payload),nil
}