package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"strconv"
	"sync"
	"time"
)

// --------------- AppCmd ---------------

// AppCmd returns the cobra command for APP
func testCmd() *cobra.Command {
	return testStartCmd
}

var (
	testStartCmd = &cobra.Command{
		Use:   "test",
		Short: "Starts to test fabric.",
		Long:  `Starts to test the network.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return testmain(args)
		},
	}
)

func testinvoke() {
	//body := `{
	//	"Id": "123"
	//}`

	//args := []string{"key", string(body)}
	////adapter := ContextMap["channel1peerorg1"]
	////_, err := adapter.Invoke(args)
	////_, err := org1channel1.Invoke(chainCodeID, args)
	//if err != nil {
	//	logger.Errorf("testinvoke Error: %s", err)
	//}
}

func testquery() {
	//args := []string{"getPolicy", "123"}
	//adapter := ContextMap["channel1peerorg1"]
	//_, err := adapter.Query(args)
	//if err != nil {
	//	logger.Errorf("testquery Error: %s", err)
	//}
}

func testmain(args []string) error {
	loops := 10

	if len(args) == 1 {
		if args[0] != "" {
			loop, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Println("loop err in args conv: ", err)
			} else {
				loops = loop
			}
		}
	}
	logger.Infof("Starting test...")
	var wg sync.WaitGroup
	wg.Add(loops)
	t1 := time.Now()
	for i := 0; i < loops; i ++ {
		fmt.Print(i)
		fmt.Print(" ")
		go func() {
			defer wg.Done()
			testinvoke()
		}()
	}
	fmt.Println()
	wg.Wait()
	elapsed := time.Since(t1)
	fmt.Println("test invoke elapsed: ", elapsed)

	wg.Add(loops)
	t1 = time.Now()
	for i := 0; i < loops; i ++ {
		fmt.Print(i)
		fmt.Print(" ")
		go func() {
			defer wg.Done()
			testquery()
		}()
	}
	fmt.Println()
	wg.Wait()
	elapsed = time.Since(t1)
	fmt.Println("test query elapsed: ", elapsed)

	return nil

}
