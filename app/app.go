package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gocraft/web"
	"github.com/spf13/viper"
	"gitlab.chainnova.com/motcert-backend/app/business"
	"gitlab.chainnova.com/motcert-backend/app/session"
	"io/ioutil"
	"net/http"
	"strings"
)

type User struct {
	Name     string
	Password string
}

type Result struct {
	ResultCode int
	Message    string
	Data       interface{}
}

type Certificate struct {
	Id string `protobuf:"bytes,1,opt,name=id" json:"id"`
}

// start app Service
func appService(args []string) error {
	// Create and register the REST service if configured
	startService()
	logger.Infof("Starting app service...")
	return nil
}

// startService initializes the REST service and adds the required
// middleware and routes.
func startService() {
	// Initialize the REST service object
	tlsEnabled := viper.GetBool("app.tls.enabled")
	address := viper.GetString("app.address")
	logger.Infof("Initializing the REST service on %s, TLS is %s.", address, (map[bool]string{true: "enabled", false: "disabled"})[tlsEnabled])
	router := buildRouter()
	startServerFinally(tlsEnabled, viper.GetString("app.address"), router)
}

func buildRouter() *web.Router {
	//parent.apps
	router := web.New(motCertAPP{})

	// Add middleware
	router.Middleware((*motCertAPP).setResponseType)
	//router.Middleware((*motCertAPP).basicAuthenticate)

	app := router.Subrouter(motCertAPP{}, "/v1/")
	app.Post("login", (*motCertAPP).postLogin)
	app.Post("certificate", (*motCertAPP).postCertificate)
	app.Get("certificate/:id", (*motCertAPP).getCertificate)
	app.Post("logout", (*motCertAPP).postLogout)

	return router
}

// defines the REST service object.
type motCertAPP struct {
}

// setResponseType is a middleware function that sets the appropriate response
// headers. Currently, it is setting the "Content-Type" to "application/json" as
// well as the necessary headers in order to enable CORS for Swagger usage.
func (s *motCertAPP) setResponseType(rw web.ResponseWriter, req *web.Request, next web.NextMiddlewareFunc) {
	rw.Header().Set("Content-Type", "application/json")

	// Enable CORS
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Access-Control-Allow-Headers", "accept, content-type")

	next(rw, req)
}

// basicAuthenticate basic authentication
func (s *motCertAPP) basicAuthenticate(rw web.ResponseWriter, req *web.Request, next web.NextMiddlewareFunc) {
	const basicScheme string = "Basic "

	// Confirm the request is sending Basic Authentication credentials.
	auth := req.Header.Get("Authorization")
	if !strings.HasPrefix(auth, basicScheme) {
		logger.Errorf("authentication error: scheme=%v", auth)
		return
	}

	// Get the plain-text username and password from the request.
	// The first six characters are skipped - e.g. "Basic ".
	str, err := base64.StdEncoding.DecodeString(auth[len(basicScheme):])
	if err != nil {
		logger.Errorf("authentication error: auth=%v", str)
		return
	}

	// Split on the first ":" character only, with any subsequent colons assumed to be part
	// of the password. Note that the RFC2617 standard does not place any limitations on
	// allowable characters in the password.
	creds := bytes.SplitN(str, []byte(":"), 2)

	if len(creds) != 2 {
		logger.Errorf("authentication error: creds=%v", creds)
		return
	}

	user := string(creds[0])
	pass := string(creds[1])

	// TODO: check user and pass

	// Set header for later use
	req.Header.Set("user", user)
	req.Header.Set("pass", pass)
	logger.Infof("basic authentication: user=%v, pass=%v", user, pass)

	next(rw, req)
}

func (s *motCertAPP) postLogin(rw web.ResponseWriter, req *web.Request) {
	logger.Info("postLogin")
	sess := session.GlobalSessions.SessionStart(rw, req.Request)
	encoder := json.NewEncoder(rw)
	var result Result
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		deal4xx(result, encoder, err, rw, 400)
		return
	}
	var user User
	err = json.Unmarshal(body, &user)
	if err != nil {
		deal4xx(result, encoder, err, rw, 400)
		return
	}
	logger.Infof("login: user=%v, pass=%v", user.Name, user.Password)
	if isRightUser(user) {
		rw.WriteHeader(http.StatusOK)
		result.ResultCode = http.StatusOK
		result.Message = "login in"
		result.Data = ""
		if err := sess.Set("username", user.Name); err != nil {
			logger.Fatal("set session username error: %v", err)
		}
		if err := encoder.Encode(result); err != nil {
			logger.Fatalf("serializing result: %v", err)
		}
	} else {
		deal4xx(result, encoder, nil, rw, http.StatusForbidden)
	}
	return
}

func isRightUser(user User) bool {
	switch user.Name {
	case "adminuser1":
		if user.Password == "XSJXSvdHN9" {
			return true
		}
		logger.Info("adminuser1 pass")
		break
	case "adminuser2":
		if user.Password == "b4Dl6XhbB" {
			return true
		}
		logger.Info("adminuser2 pass")
		break
	default:
		return false
	}
	return false
}

func (s *motCertAPP) postLogout(rw web.ResponseWriter, req *web.Request) {
	logger.Info("postLogout")
	sess := session.GlobalSessions.SessionStart(rw, req.Request)
	if err := sess.Delete("username"); err != nil {
		logger.Fatal("delete session username error: %v", err)
	}
	session.GlobalSessions.SessionDestroy(rw, req.Request)
	var result Result
	if isLogin(rw, req) {
		result.ResultCode = http.StatusBadRequest
		result.Message = "logout failed"
		logger.Error("Should not be here default")
	} else {
		result.ResultCode = http.StatusOK
		result.Message = "logout success"
	}

	encoder := json.NewEncoder(rw)
	if err := encoder.Encode(result); err != nil {
		logger.Fatalf("serializing result: %v", err)
	}
	return
}

func isLogin(rw web.ResponseWriter, req *web.Request) bool {
	sess := session.GlobalSessions.SessionStart(rw, req.Request)
	currentUser := sess.Get("username")
	switch currentUser {
	case nil:
		return false
	case "":
		return false
	default:
		logger.Error("Should not be here default")
		return true
	}
}

func deal4xx(result Result, encoder *json.Encoder, err error, rw web.ResponseWriter, code int) {
	resultCode := http.StatusBadRequest
	if code > 0 {
		resultCode = code
	}
	result.ResultCode = resultCode
	if err != nil {
		result.Message = err.Error()
	}
	rw.WriteHeader(resultCode)
	if err := encoder.Encode(result); err != nil {
		logger.Fatalf("serializing result: %v", err)
	}
	logger.Errorf("Error: %s  :read request error", err)
}

func (s *motCertAPP) postCertificate(rw web.ResponseWriter, req *web.Request) {
	encoder := json.NewEncoder(rw)
	var result Result
	if isLogin(rw, req) {
		// Decode the incoming JSON payload
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			deal4xx(result, encoder, err, rw, 400)
			return
		}

		var certificate Certificate
		err = json.Unmarshal(body, &certificate)
		if err != nil {
			deal4xx(result, encoder, err, rw, 400)
			return
		}

		logger.Infof("postCertificate: '%s'.\n", certificate)
		args := []string{string(body)}
		txID, err := business.CertificateIn(FabricSetupEntity, args)
		if err != nil {
			result.ResultCode = http.StatusNotImplemented
			result.Message = err.Error()
			rw.WriteHeader(http.StatusNotImplemented)
			if err := encoder.Encode(result); err != nil {
				logger.Fatalf("serializing result: %v", err)
			}
			logger.Errorf("Error: %s", err)
			return
		}

		result.ResultCode = http.StatusOK
		rw.WriteHeader(http.StatusOK)
		result.Message = fmt.Sprintf("%v", txID)
		logger.Infof("postPolicy: '%s'\n", txID)

	} else {
		result.ResultCode = http.StatusNetworkAuthenticationRequired
		result.Message = "Should login"
	}

	if err := encoder.Encode(result); err != nil {
		logger.Fatalf("serializing result: %v", err)
	}
	return
}

func (s *motCertAPP) getCertificate(rw web.ResponseWriter, req *web.Request) {
	encoder := json.NewEncoder(rw)
	var result Result

	id := req.PathParams["id"]
	args := []string{id}
	response, err := business.CertificateOut(FabricSetupEntity, args)
	if err != nil {
		deal4xx(result, encoder, err, rw, 400)
		return
	}

	if response == "" {
		result.ResultCode = http.StatusBadRequest
		result.Message = "have not such data"
		rw.WriteHeader(http.StatusBadRequest)
		if err := encoder.Encode(result); err != nil {
			logger.Fatalf("serializing result: %v", err)
		}
		logger.Infof("getCertificate: '%s'\n", "")
		return
	}

	var certificateReq Certificate
	err = json.Unmarshal([]byte(response), &certificateReq)
	if err != nil {
		result.ResultCode = http.StatusNotImplemented
		result.Message = err.Error()
		rw.WriteHeader(http.StatusNotImplemented)
		if err := encoder.Encode(result); err != nil {
			logger.Fatalf("serializing result: %v", err)
		}
		logger.Errorf("Error: %s", err)
		return
	}

	var certificate Certificate
	certificate = Certificate{
		Id: certificateReq.Id,
	}

	result.Data = certificate

	if err := encoder.Encode(result); err != nil {
		logger.Fatalf("serializing result: %v", err)
	}
	logger.Infof("getPolicy: '%s'\n", response)

	return
}

/**
http://www.360doc.com/content/16/0709/16/478627_574275652.shtml

guolidong:~$ openssl genrsa -out ca.key 2048
Generating RSA private key, 2048 bit long modulus
...+++
.......................+++
unable to write 'random state'
e is 65537 (0x10001)
guolidong:~$ openssl req -x509 -new -nodes -key ca.key -subj "/CN=cert.mot.gov.cn" -days 5000 -out ca.crt
guolidong:~$ openssl genrsa -out server.key 2048
Generating RSA private key, 2048 bit long modulus
............+++
................+++
unable to write 'random state'
e is 65537 (0x10001)
guolidong:~$ openssl req -new -key server.key -subj "/CN=cert.mot.gov.cn" -out server.csr
guolidong:~$ openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 5000
Signature ok
subject=/CN=cert.mot.gov.cn
Getting CA Private Key
unable to write 'random state'
guolidong:~$
guolidong:~$
guolidong:~$ openssl rsa -in server.key -out server.key.public
writing RSA key
guolidong:~$ openssl genrsa -out client.key 2048
Generating RSA private key, 2048 bit long modulus
..+++
..................................................................................................................................+++
unable to write 'random state'
e is 65537 (0x10001)
guolidong:~$ openssl req -new -key client.key -subj "/CN=cert.mot.gov.cn" -out client.csr
guolidong:~$ openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt -days 5000
Signature ok
subject=/CN=cert.mot.gov.cn
Getting CA Private Key
unable to write 'random state'

openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -extfile client.ext -out client.crt -days 5000

guolidong:~$ cat client.crt client.key> client.pem
guolidong:~$ cat server.crt server.key > server.pem
guolidong:~$ openssl pkcs12 -export -inkey client.key -in client.crt -out client.pfx
Enter Export Password:
Verifying - Enter Export Password:
unable to write 'random state'
guolidong:~$ openssl pkcs12 -export -inkey server.key -in server.crt -out server.pfx
Enter Export Password:
Verifying - Enter Export Password:
unable to write 'random state'
 */
func startServerFinally(tlsEnabled bool, currentAddress string, router *web.Router) {

	pool := x509.NewCertPool()
	caCertPath := viper.GetString("app.tls.ca.file")

	caCrt, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		logger.Error("ReadFile err:", err)
		return
	}
	pool.AppendCertsFromPEM(caCrt)

	s := &http.Server{
		Addr:    currentAddress,
		Handler: router,
		TLSConfig: &tls.Config{
			ClientCAs:  pool,
			ClientAuth: tls.RequireAndVerifyClientCert,
			MaxVersion: tls.VersionTLS12,
			MinVersion: tls.VersionTLS10,
		},
	}

	// Start server
	if tlsEnabled {
		err := s.ListenAndServeTLS(viper.GetString("app.tls.cert.file"), viper.GetString("app.tls.key.file"))
		if err != nil {
			logger.Errorf("ListenAndServeTLS: %s", err)
		}
	} else {
		err := http.ListenAndServe(currentAddress, router)
		if err != nil {
			logger.Errorf("ListenAndServe: %s", err)
		}
	}
}
