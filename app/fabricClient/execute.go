package fabricClient

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/pkg/errors"
	"time"
)

func (setup *FabricSetup) Execute(eventID, methodName string, args []string) (string, error) {

	transientDataMap := make(map[string][]byte)
	transientDataMap["result"] = []byte("Transient data in mot cert execute ...")

	//Register a notification handler on the client
	reg, notifier, err := setup.event.RegisterChaincodeEvent(setup.ChainCodeID, eventID)
	if err != nil {
		return "", err
	}
	defer setup.event.Unregister(reg)

	var paraArgs [][]byte
	for _, arg := range args {
		paraArgs = append(paraArgs, []byte(arg))
	}
	// Create a request (proposal) and send it
	response, err := setup.client.Execute(channel.Request{ChaincodeID: setup.ChainCodeID, Fcn: methodName, Args: paraArgs, TransientMap: transientDataMap})
	if err != nil {
		return "", errors.Errorf("failed to execute: %v", err)
	}

	// Wait for the result of the submission
	select {
	case ccEvent := <-notifier:
		logger.Infof("Received CC event: %s\n", ccEvent)
	case <-time.After(time.Second * 30):
		return "", errors.Errorf("did NOT receive CC event for eventId(%s)", eventID)
	}

	return string(response.TransactionID), nil
}
