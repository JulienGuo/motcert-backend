package fabricClient

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/pkg/errors"
	"time"
)

func (setup *FabricSetup) Execute(eventID, methodName string, args []string) (string, error) {

	transientDataMap := make(map[string][]byte)
	transientDataMap["result"] = []byte("Transient data in mot cert execute ...")

	logger.Info("-----Execute-----")
	//Register a notification handler on the client
	reg, notifier, err := setup.event.RegisterChaincodeEvent(setup.ChainCodeID, eventID)
	logger.Info("-----Execute 2-----")
	if err != nil {
		return "", err
	}
	defer setup.event.Unregister(reg)

	logger.Info("-----Execute 3-----")
	var paraArgs [][]byte
	paraArgs = append(paraArgs, []byte(methodName))
	for _, arg := range args {
		paraArgs = append(paraArgs, []byte(arg))
	}
	logger.Info("-----Execute 4-----")
	// Create a request (proposal) and send it
	response, err := setup.client.Execute(channel.Request{ChaincodeID: setup.ChainCodeID, Fcn: "invoke", Args: paraArgs, TransientMap: transientDataMap})
	if err != nil {
		logger.Info("-----Execute 5-----")
		return "", errors.Errorf("failed to execute: %v", err)
	}

	// Wait for the result of the submission
	select {
	case ccEvent := <-notifier:
		logger.Infof("Received CC event: %s\n", ccEvent)
	case <-time.After(time.Second * 30):
		logger.Info("-----Execute 6-----")
		return "", errors.Errorf("did NOT receive CC event for eventId(%s)", eventID)
	}

	return string(response.TransactionID), nil
}
