package fabricClient

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/pkg/errors"
)

func (setup *FabricSetup) Query(methodName string,args []string) (string, error) {

	logger.Error(args,"|||||")
	//Prepare arguments
	var paraArgs [][]byte
	for _, arg := range args {
		paraArgs = append(paraArgs, []byte(arg))
	}
	logger.Error(paraArgs,"|||||")

	response, err := setup.client.Query(channel.Request{ChaincodeID: setup.ChainCodeID, Fcn: methodName, Args: paraArgs})
	if err != nil {
		return "", errors.Errorf("failed to query :%v", err)
	}
	return string(response.Payload), nil
}
