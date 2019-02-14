package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/gocraft/web"
	"github.com/spf13/viper"
	"gitlab.chainnova.com/motcert-backend/app/business"
	"gitlab.chainnova.com/motcert-backend/app/session"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

type User struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type Result struct {
	ResultCode int         `json:"resultCode"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data"`
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
	app.Post("uploadFile", (*motCertAPP).postUploadFile)
	app.Get("downloadFile/:certId", (*motCertAPP).getDownloadFile)
	app.Get("certificate/:certId", (*motCertAPP).getCertificate)
	app.Post("certificate/openList", (*motCertAPP).getOpenList)
	app.Post("certificate/deletedList", (*motCertAPP).getDeletedList)
	app.Post("certificate/draftList", (*motCertAPP).getDraftList)
	app.Post("openStatus", (*motCertAPP).postOpenStatus)
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
	logger.Infof("basic authentication: user=%v", user)

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
	logger.Infof("postCertificate start")
	encoder := json.NewEncoder(rw)
	var result Result
	if isLogin(rw, req) {
		// Decode the incoming JSON payload
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			deal4xx(result, encoder, err, rw, 400)
			return
		}

		data, err, code := business.CertificateIn(FabricSetupEntity, body)
		if err != nil {
			deal4xx(result, encoder, err, rw, code)
			return
		}

		result.ResultCode = code
		rw.WriteHeader(code)
		result.Data = data
		result.Message = "OK"
	} else {
		result.ResultCode = http.StatusUnauthorized
		result.Message = "Should login"
	}

	if err := encoder.Encode(result); err != nil {
		logger.Fatalf("serializing result: %v", err)
	}
	logger.Infof("postCertificate end")
	return
}

func (s *motCertAPP) postUploadFile(rw web.ResponseWriter, req *web.Request) {
	logger.Infof("postUploadFile start")
	encoder := json.NewEncoder(rw)
	var result Result
	if isLogin(rw, req) {
		err := req.ParseMultipartForm(10 << 20)
		if err != nil {
			deal4xx(result, encoder, err, rw, 400)
			return
		}

		file, handler, err := req.FormFile("certFile")
		if err != nil {
			deal4xx(result, encoder, err, rw, http.StatusBadRequest)
			return
		}
		defer closeFile(file)
		certId := req.MultipartForm.Value["certId"][0]
		fileName := "./tempUploadFiles/" + certId + handler.Filename
		deleteFileOnDisk(fileName)
		f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			deal4xx(result, encoder, err, rw, http.StatusInternalServerError)
			return
		}
		defer closeFile(f)

		written, err := io.Copy(f, file)
		if err != nil {
			deal4xx(result, encoder, err, rw, http.StatusInternalServerError)
			return
		}

		data, err, code := business.UploadFile(FabricSetupEntity, certId, fileName)
		if err != nil {
			deal4xx(result, encoder, err, rw, code)
			return
		}
		result.ResultCode = code
		rw.WriteHeader(code)
		result.Data = data

		fileName2 := "../files/" + certId + handler.Filename
		deleteFileOnDisk(fileName2)
		f2, err := os.OpenFile(fileName2, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			deal4xx(result, encoder, err, rw, http.StatusInternalServerError)
			return
		}
		defer closeFile(f2)

		err = os.Rename(fileName, fileName2)
		if err != nil {
			deal4xx(result, encoder, err, rw, http.StatusInternalServerError)
		}

		str := strconv.FormatInt(written, 10)
		result.Message = "uploaded=" + str
	} else {
		result.ResultCode = http.StatusUnauthorized
		result.Message = "Should login"
	}

	if err := encoder.Encode(result); err != nil {
		logger.Fatalf("serializing result: %v", err)
	}
	logger.Infof("postUploadFile end")
	return
}

func deleteFileOnDisk(localPath string) {
	logger.Debugf("remove file: %s", localPath)
	if err := os.Remove(localPath); err != nil {
		logger.Error(err)
	}
	dirsList := make([]string, 0, 0)
	for dir := path.Dir(localPath); dir != "../tempUploadFiles" && len(dir) > len("../tempUploadFiles"); dir = path.Dir(dir) {
		dirsList = append(dirsList, dir)
	}
	sort.StringSlice(dirsList).Sort()
	for i := len(dirsList) - 1; i >= 0; i-- {
		f, err := os.Open(dirsList[i])
		if err != nil {
			logger.Error(err)
		}
		fs, err2 := f.Readdirnames(1)
		if err2 == io.EOF && (fs == nil || len(fs) == 0) {
			closeFile(f)
			logger.Debugf("remove dir: %s", dirsList[i])
			if err := os.Remove(dirsList[i]); err != nil {
				logger.Error(err)
			}
			continue
		} else if err2 != nil {
			logger.Error(err2)
		}
		closeFile(f)
	}

}

func (s *motCertAPP) getDownloadFile(rw web.ResponseWriter, req *web.Request) {
	logger.Infof("getDownloadFile start")
	encoder := json.NewEncoder(rw)
	var result Result

	//defer closeFile(file)
	//certId := req.MultipartForm.Value["certId"][0]
	//fileName := "../files/" + certId + handler.Filename
	fileFullPath := "../files/" + "GDQ2018-223-001SCAN0021.pdf"
	file, err := os.Open(fileFullPath)
	if err != nil {
		deal4xx(result, encoder, err, rw, http.StatusInternalServerError)
		return
	}
	defer closeFile(file)

	fileName := path.Base(fileFullPath)
	fileName = url.QueryEscape(fileName) // 防止中文乱码
	rw.Header().Add("Content-Type", "application/octet-stream")
	rw.Header().Add("content-disposition", "attachment; filename=\""+fileName+"\"")
	_, error := io.Copy(rw, file)
	if error != nil {
		deal4xx(result, encoder, err, rw, http.StatusInternalServerError);
		return
	}

	select {
	case connectEnd := <-rw.CloseNotify():
		logger.Info("copy end")
		if connectEnd {
			deal4xx(result, encoder, errors.New("client connection has gone away"), rw, http.StatusBadRequest)
			return
		}
	case <-time.After(time.Second * 120):
		logger.Info("copy timeout")
	}

	//data, err, code := business.UploadFile(FabricSetupEntity, certId, fileName)
	//if err != nil {
	//	deal4xx(result, encoder, err, rw, code)
	//	return
	//}
	result.ResultCode = 200
	rw.WriteHeader(200)
	result.Data = nil

	result.Message = "download"

	if err := encoder.Encode(result); err != nil {
		logger.Fatalf("serializing result: %v", err)
	}
	logger.Infof("getDownloadFile end")
	return
}

//func getDownloadFile(fileFullPath string, res *restful.Response) {
//	file, err := os.Open(fileFullPath)
//
//	if err != nil {
//		res.WriteEntity(_dto.ErrorDto{Err: err})
//		return
//	}
//
//	defer file.Close()
//	fileName := path.Base(fileFullPath)
//	fileName = url.QueryEscape(fileName) // 防止中文乱码
//	res.AddHeader("Content-Type", "application/octet-stream")
//	res.AddHeader("content-disposition", "attachment; filename=\""+fileName+"\"")
//	_, error := io.Copy(res.ResponseWriter, file)
//	if error != nil {
//		res.WriteErrorString(http.StatusInternalServerError, err.Error())
//		return
//	}
//}

func closeFile(f multipart.File) {
	err := f.Close()
	if err != nil {
		logger.Error(err)
		return
	}
}

func (s *motCertAPP) getCertificate(rw web.ResponseWriter, req *web.Request) {
	logger.Infof("getCertificate start")
	encoder := json.NewEncoder(rw)
	var result Result
	data, err, code := business.CertificateOut(FabricSetupEntity, &req.PathParams)
	if err != nil {
		deal4xx(result, encoder, err, rw, code)
		return
	}

	result.ResultCode = code
	result.Message = "OK"

	result.Data = data

	if err := encoder.Encode(result); err != nil {
		logger.Fatalf("serializing result: %v", err)
	}

	logger.Infof("getCertificate end")
	return
}

func (s *motCertAPP) getOpenList(rw web.ResponseWriter, req *web.Request) {
	logger.Infof("getOpenList start")
	encoder := json.NewEncoder(rw)
	var result Result
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		deal4xx(result, encoder, err, rw, 400)
		return
	}

	data, err, code := business.OpenListRichQuery(FabricSetupEntity, body, isLogin(rw, req))
	if err != nil {
		deal4xx(result, encoder, err, rw, code)
		return
	}
	result.ResultCode = http.StatusOK
	result.Message = "OK"
	result.Data = data

	if err := encoder.Encode(result); err != nil {
		logger.Fatalf("serializing result: %v", err)
	}

	logger.Infof("getOpenList end")
	return
}

func (s *motCertAPP) getDeletedList(rw web.ResponseWriter, req *web.Request) {
	logger.Infof("getDeletedList start")
	encoder := json.NewEncoder(rw)
	var result Result
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		deal4xx(result, encoder, err, rw, 400)
		return
	}

	data, err, code := business.DeletedListRichQuery(FabricSetupEntity, body, isLogin(rw, req))
	if err != nil {
		deal4xx(result, encoder, err, rw, code)
		return
	}
	result.ResultCode = http.StatusOK
	result.Message = "OK"
	result.Data = data

	if err := encoder.Encode(result); err != nil {
		logger.Fatalf("serializing result: %v", err)
	}

	logger.Infof("getDeletedList end")
	return
}

func (s *motCertAPP) getDraftList(rw web.ResponseWriter, req *web.Request) {
	logger.Infof("getDraftList start")
	encoder := json.NewEncoder(rw)
	var result Result
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		deal4xx(result, encoder, err, rw, 400)
		return
	}

	data, err, code := business.DraftListRichQuery(FabricSetupEntity, body, isLogin(rw, req))
	if err != nil {
		deal4xx(result, encoder, err, rw, code)
		return
	}
	result.ResultCode = http.StatusOK
	result.Message = "OK"
	result.Data = data

	if err := encoder.Encode(result); err != nil {
		logger.Fatalf("serializing result: %v", err)
	}

	logger.Infof("getDraftList end")
	return
}

func (s *motCertAPP) postOpenStatus(rw web.ResponseWriter, req *web.Request) {
	logger.Infof("postOpenStatus start")
	//先查询再更改
	encoder := json.NewEncoder(rw)
	var result Result
	if isLogin(rw, req) {
		// Decode the incoming JSON payload
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			deal4xx(result, encoder, err, rw, 400)
			return
		}

		data, err, code := business.ChangeStatus(FabricSetupEntity, body)

		if err != nil {
			deal4xx(result, encoder, err, rw, code)
			return
		}
		result.ResultCode = code
		rw.WriteHeader(code)
		result.Data = data
	} else {
		result.ResultCode = http.StatusUnauthorized
		result.Message = "Should login"
	}

	if err := encoder.Encode(result); err != nil {
		logger.Fatalf("serializing result: %v", err)
	}
	logger.Infof("postOpenStatus end")
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
