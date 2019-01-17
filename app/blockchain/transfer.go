package blockchain

import (
	"time"
	"math/rand"
	"strconv"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"fmt"
	"io/ioutil"
	"paillier"
)

//// 将转账金额传入，并且随机数传入r1,r2   chaincode端进行接受 A , B , X , r1 ,r2 ,pubByteAKey ,pubByteBKey
func (setup *FabricSetup)Transfer(userA string,userB string,cash string,addressA string,addressB string)(string,error){

	//生成两个随机数，传递到chaincode当中
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	pubByteUser , err := ioutil.ReadFile("public"+userA+".pem")
	if err != nil {
		return "",err
	}
	pubKey, err := paillier.ParsePublicKey(pubByteUser)
	if err != nil {
		return "", err
	}

	length := pubKey.N.BitLen()
	r1 := r.Intn(length)
	r2 := r.Intn(length)

	r1Str := strconv.Itoa(r1)
	r2Str := strconv.Itoa(r2)

	fmt.Println(" r1: ",r1Str," r2 :",r2Str)
	//Prepare arguments
	var args []string
	args = append(args,"invoke")
	args = append(args,"transfer")
	args = append(args,userA)
	args = append(args,userB)
	args = append(args,cash)
	args = append(args,r1Str)
	args = append(args,r2Str)
	args = append(args,addressA)
	args = append(args,addressB)

	eventID := "transferEvent"

	// Add data that will be visible in the proposal, like a description of the invoke request
	transientDataMap := make(map[string][]byte)
	transientDataMap["result"] = []byte("Transient data in transfer invoke")

	// Register a notification handler on the client
	reg , notifier ,err :=setup.event.RegisterChaincodeEvent(setup.ChainCodeID,eventID)

	if err != nil {
		return "", err
	}

	defer setup.event.Unregister(reg)

	// Create a request (proposal) and send it
	response, err := setup.client.Execute(channel.Request{ChaincodeID: setup.ChainCodeID, Fcn: args[0],
	Args: [][]byte{[]byte(args[1]), []byte(args[2]), []byte(args[3]),[]byte(args[4]),[]byte(args[5]),[]byte(args[6]),[]byte(args[7]),[]byte(args[8])}, TransientMap: transientDataMap})
	if err != nil {
		return "", fmt.Errorf("failed to move funds  (Transfer): %v", err)
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