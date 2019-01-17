package blockchain

import (
	"math/big"
	"paillier"
	"io/ioutil"
	"encoding/hex"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"fmt"
	"time"
)

func(setup *FabricSetup)InitParam(user1 string,user1Value int64 ,user2 string,user2Value int64) (string,error) {

	fmt.Println("-----plainA-----",user1Value)
	fmt.Println("-----plianB-----",user2Value)
	//Prepare arguments
	user1ValueB := new(big.Int).SetInt64(user1Value)
	user2ValueB := new(big.Int).SetInt64(user2Value)

	pubByteUser1 , err := ioutil.ReadFile("public"+user1+".pem")
	if err != nil {
		return "",err
	}
	pubByteUser2 , err := ioutil.ReadFile("public"+user2+".pem")
	if err != nil {
		return "",err
	}

	ciperUser1Value , err := paillier.Encrypt(pubByteUser1,user1ValueB.Bytes())
	if err != nil {
		return "",err
	}
	ciperUser1ValueStr := hex.EncodeToString(ciperUser1Value)


	ciperUser2Value , err := paillier.Encrypt(pubByteUser2,user2ValueB.Bytes())
	if err != nil {
		return "",err
	}
	ciperUser2ValueStr := hex.EncodeToString(ciperUser2Value)


	fmt.Println("-------------ciperUser1Val-----------",ciperUser1ValueStr)
	fmt.Println("--------------ciperUser2Val----------",ciperUser2ValueStr)

	var args []string
	args = append(args,"invoke")
	args = append(args,"initParam")
	args = append(args,user1)
	args = append(args,ciperUser1ValueStr)
	args = append(args,user2)
	args = append(args,ciperUser2ValueStr)

	eventID := "initParamEvent"

	// Add data that will be visible in the proposal, like a description of the invoke request
	transientDataMap := make(map[string][]byte)
	transientDataMap["result"] = []byte("Transient data in initParam")

	// Register a notification handler on the client
	reg , notifier ,err :=setup.event.RegisterChaincodeEvent(setup.ChainCodeID,eventID)

	if err != nil {
		return "", err
	}
	defer setup.event.Unregister(reg)


	// Create a request (proposal) and send it
	// Create a request (proposal) and send it
	response, err := setup.client.Execute(channel.Request{ChaincodeID: setup.ChainCodeID, Fcn: args[0], Args: [][]byte{[]byte(args[1]), []byte(args[2]), []byte(args[3]),[]byte(args[4]),[]byte(args[5])}, TransientMap: transientDataMap})
	if err != nil {
		return "", fmt.Errorf("failed to move funds  (Init Param): %v", err)
	}

	// Wait for the result of the submission
	select {
	case ccEvent := <-notifier:
		fmt.Printf("Received CC event: %s\n", ccEvent)
	case <-time.After(time.Second * 20):
		return "", fmt.Errorf("did NOT receive CC event for eventId(%s)", eventID)
	}


	return string(response.TransactionID), nil

}
