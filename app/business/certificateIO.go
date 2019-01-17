package business

import (
	"github.com/op/go-logging"
	"gitlab.chainnova.com/motcert-backend/app/fabricClient"
)

var logger = logging.MustGetLogger("Motcert.business")

func CertificateIn(setup *fabricClient.FabricSetup, args []string) (string, error) {

	logger.Info("-----CertificateIn-----")
	eventID := "postCertificateEvent"
	return setup.Execute(eventID, "postCertificate", args)
}

func CertificateOut(setup *fabricClient.FabricSetup, args []string) (string, error) {

	logger.Info("-----------------------------CertificateOut BEGIN---------------------")

	var paraArgs []string
	paraArgs = append(paraArgs, "getCertificate")
	for _, arg := range args {
		paraArgs = append(paraArgs, arg)
	}

	return setup.Query(paraArgs)
}
