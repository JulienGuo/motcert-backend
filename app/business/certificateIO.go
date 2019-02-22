package business

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/op/go-logging"
	"gitlab.chainnova.com/motcert-backend/app/fabricClient"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

var logger = logging.MustGetLogger("Motcert.business")

var openListBookmarks []string
var deletedListBookmarks []string
var draftListBookmarks []string

func init() {
	//可以加定时任务优化
	openListBookmarks = append(openListBookmarks, "")
	deletedListBookmarks = append(deletedListBookmarks, "")
	draftListBookmarks = append(draftListBookmarks, "")
}

type Certificate struct {
	CertId              string `protobuf:"bytes,1,req,name=certId" json:"certId"`                            //证书编号
	CertType            string `protobuf:"bytes,2,req,name=certType" json:"certType"`                        //证书类型
	EntrustOrg          string `protobuf:"bytes,3,opt,name=entrustOrg" json:"entrustOrg"`                    //委托单位
	InstrumentName      string `protobuf:"bytes,4,opt,name=instrumentName" json:"instrumentName"`            //器具名称
	Spec                string `protobuf:"bytes,5,opt,name=spec" json:"spec"`                                //型号/规格
	ExportId            string `protobuf:"bytes,6,opt,name=exportId" json:"exportId"`                        //出厂编号
	MadeByOrg           string `protobuf:"bytes,7,opt,name=madeByOrg" json:"madeByOrg"`                      //制造单位
	EntrustOrgAdd       string `protobuf:"bytes,8,opt,name=entrustOrgAdd" json:"entrustOrgAdd"`              //委托单位地址
	Approver            string `protobuf:"bytes,9,opt,name=approver" json:"approver"`                        //批准人
	Verifier            string `protobuf:"bytes,10,opt,name=verifier" json:"verifier"`                       //核验员
	CalibratePerson     string `protobuf:"bytes,11,opt,name=calibratePerson" json:"calibratePerson"`         //校准员
	CalibrateDate       string `protobuf:"bytes,12,opt,name=calibrateDate" json:"calibrateDate"`             //校准日期
	SuggestNextCaliDate string `protobuf:"bytes,13,opt,name=suggestNextCaliDate" json:"suggestNextCaliDate"` //建议下次校准日期
	IsCompleted         bool   `protobuf:"bytes,14,opt,name=isCompleted" json:"isCompleted"`                 //是否完成
	IsOpen              bool   `protobuf:"bytes,15,opt,name=isOpen" json:"isOpen"`                           //是否公开
	IsDeleted           bool   `protobuf:"bytes,16,opt,name=isDeleted" json:"isDeleted"`                     //是否删除
	UpdateDate          string `protobuf:"bytes,17,opt,name=updateDate" json:"updateDate"`                   //最新修改日期
	HasUpload           bool   `protobuf:"bytes,18,opt,name=hasUpload" json:"hasUpload"`                     //是否已上传pdf文件
}

type FileStruct struct {
	CertId   string `protobuf:"bytes,1,req,name=certId" json:"certId"`     //证书编号
	CertFile []byte `protobuf:"bytes,2,req,name=certFile" json:"certFile"` //证书pdf文件
	FileName string `protobuf:"bytes,3,req,name=fileName" json:"fileName"` //文件名
}

type Status struct {
	CertId           string `json:"certId"`
	IsOpen           bool   `json:"isOpen"`
	IsCompleted      bool   `json:"isCompleted"`
	IsDeleted        bool   `json:"isDeleted"`
	IsChangedOnChain bool   `json:"isChangedOnChain"`
}

type QueryConditions struct {
	PageSize        int    `protobuf:"bytes,1,req,name=pageSize" json:"pageSize"`                //每页数据量
	PageIndex       int    `protobuf:"bytes,2,req,name=pageIndex" json:"pageIndex"`              //本次请求的页号
	IsOpen          bool   `protobuf:"bytes,3,opt,name=isOpen" json:"isOpen"`                    //是否公开
	IsCompleted     bool   `protobuf:"bytes,4,opt,name=isCompleted" json:"isCompleted"`          //是否完成
	IsDeleted       bool   `protobuf:"bytes,5,opt,name=isDeleted" json:"isDeleted"`              //是否删除
	CertType        string `protobuf:"bytes,6,opt,name=certType" json:"certType"`                //证书类型
	CertId          string `protobuf:"bytes,7,opt,name=certId" json:"certId"`                    //证书编号
	EntrustOrg      string `protobuf:"bytes,8,opt,name=entrustOrg" json:"entrustOrg"`            //委托单位
	InstrumentName  string `protobuf:"bytes,9,opt,name=instrumentName" json:"instrumentName"`    //器具名称
	StartCreateDate string `protobuf:"bytes,10,opt,name=startCreateDate" json:"startCreateDate"` //起始录入日期
	EndCreateDate   string `protobuf:"bytes,11,opt,name=endCreateDate" json:"endCreateDate"`     //结束录入日期
	StartCalibDate  string `protobuf:"bytes,12,opt,name=startCalibDate" json:"startCalibDate"`   //起始校准日期
	EndCalibDate    string `protobuf:"bytes,13,opt,name=endCalibDate" json:"endCalibDate"`       //结束校准日期
}

type ListInternal struct {
	PageCount int
	Bookmark  string
	Certs     []Certificate
}
type List struct {
	PageCount int           `json:"pageCount"`
	PageIndex int           `json:"pageIndex"`
	Certs     []Certificate `json:"certs"`
}

func CertificateIn(setup *fabricClient.FabricSetup, body []byte) (interface{}, error, int) {

	logger.Info("-----CertificateIn BEGIN-----")
	var certificate Certificate
	err := json.Unmarshal(body, &certificate)
	if err != nil {
		return nil, err, http.StatusBadRequest
	}

	logger.Infof("postCertificate: '%s'.\n", certificate)

	args1 := []string{certificate.CertId}
	response, err := setup.Query("getDownloadFile", args1)

	if err != nil {
		certificate.HasUpload = false
	}

	if response == "" {
		certificate.HasUpload = false
	}

	var fileStruct *FileStruct
	err = json.Unmarshal([]byte(response), fileStruct)
	if err == nil && fileStruct != nil {
		certificate.HasUpload = true
	} else {
		certificate.HasUpload = false
	}

	if certificate.IsOpen && !certificate.HasUpload {
		return nil, err, http.StatusBadRequest
	}

	args2 := []string{string(body)}
	eventID := "postCertificateEvent" + certificate.CertId
	data, err := setup.Execute(eventID, "postCertificate", args2)
	if err != nil {
		return nil, err, http.StatusNotImplemented
	}
	logger.Info("-----CertificateIn END-----")
	return data, nil, http.StatusOK
}

func CertificateOut(setup *fabricClient.FabricSetup, param *map[string]string) (interface{}, error, int) {

	logger.Info("-----------------------------CertificateOut BEGIN---------------------")
	certId := make(map[string]string)
	certId = *param
	args := []string{certId["certId"]}
	response, err := setup.Query("getCertificate", args)

	if err != nil {
		return nil, err, http.StatusBadRequest
	}

	if response == "" {
		return nil, err, http.StatusNotFound
	}

	var certificate Certificate
	err = json.Unmarshal([]byte(response), &certificate)
	if err != nil {
		return nil, err, http.StatusNotImplemented
	}
	logger.Info("-----------------------------CertificateOut END---------------------")
	return certificate, nil, http.StatusOK
}

func UploadFile(setup *fabricClient.FabricSetup, certId string, file *multipart.File) (interface{}, error, int) {

	logger.Info("-----------------------------UploadFile BEGIN---------------------")

	certFullPath := "./tempUploadFiles/" + certId + ".pdf"
	deleteFileOnDisk(certFullPath)
	f, err := os.OpenFile(certFullPath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}
	defer closeFile(f)
	defer closeFile(*file)

	_, err = io.Copy(f, *file)
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}

	certf, err := os.Open(certFullPath)
	if err != nil {
		return nil, err, http.StatusNotImplemented
	}

	defer closeFile(certf)
	fileInfo, err := os.Stat(certFullPath)
	if err != nil {
		return nil, err, http.StatusNotImplemented
	}
	fileSize := fileInfo.Size()
	buf := make([]byte, fileSize)
	n, err := certf.Read(buf)
	if err != nil {
		return nil, err, http.StatusNotImplemented
	}
	var txid string
	if int64(n) == fileSize {

		fileType := http.DetectContentType(buf)
		if fileType != "application/pdf" {
			return nil, err, http.StatusBadRequest
		}

		fileStruct := FileStruct{
			CertId:   certId,
			CertFile: buf,
		}

		fs, err := json.Marshal(fileStruct)

		args := []string{string(fs)}

		eventID := "postUploadFileEvent" + certId
		txid, err = setup.Execute(eventID, "postUploadFile", args)
		if err != nil {
			return nil, err, http.StatusNotImplemented
		}

		var param map[string]string
		param = make(map[string]string)
		param["certId"] = certId
		cert, err, _ := CertificateOut(setup, &param)
		if err == nil && cert != nil {
			certificate := cert.(Certificate)

			certificate.HasUpload = true

			nerCert, err := json.Marshal(certificate)

			_, err, _ = CertificateIn(setup, nerCert)
			if err != nil {
				return nil, err, http.StatusNotImplemented
			}
		}

	} else {
		return nil, err, http.StatusNotImplemented
	}
	logger.Info("-----------------------------UploadFile END---------------------")
	return txid, nil, http.StatusOK
}

func DownloadFile(setup *fabricClient.FabricSetup, param *map[string]string) (string, string, *os.File, error, int) {

	logger.Info("-----------------------------DownloadFile BEGIN---------------------")

	certId := make(map[string]string)
	certId = *param
	certIdValue := certId["certId"]

	args := []string{certId["certId"]}
	response, err := setup.Query("getDownloadFile", args)

	if err != nil {
		return "", "", nil, err, http.StatusBadRequest
	}

	if response == "" {
		return "", "", nil, err, http.StatusNotFound
	}

	var fileStruct FileStruct
	err = json.Unmarshal([]byte(response), &fileStruct)
	if err != nil {
		return "", "", nil, err, http.StatusNotImplemented
	}

	fileFullPath := "../files/" + certIdValue + ".pdf"

	deleteFileOnDisk(fileFullPath)
	f, err := os.Create(fileFullPath)
	if err != nil {
		return "", "", nil, err, http.StatusNotImplemented
	}
	defer closeFile(f)
	_, err = f.Write(fileStruct.CertFile)
	if err != nil {
		return "", "", nil, err, http.StatusNotImplemented
	}

	file, err := os.Open(fileFullPath)

	if err != nil {
		return "", "", nil, err, http.StatusInternalServerError
	}

	fileName := path.Base(fileFullPath)
	fileName = url.QueryEscape(fileName) // 防止中文乱码

	fileInfo, err := os.Stat(fileFullPath)
	if err != nil {
		return "", "", nil, err, http.StatusInternalServerError
	}

	fileSize := strconv.FormatInt(fileInfo.Size(), 10)

	return fileName, fileSize, file, nil, http.StatusOK
}

func closeFile(f multipart.File) {
	err := f.Close()
	if err != nil {
		logger.Error(err)
		return
	}
}

func ChangeStatus(setup *fabricClient.FabricSetup, body []byte) (interface{}, error, int) {

	logger.Info("-----------------------------ChangeStatus BEGIN---------------------")
	var statuses []Status
	err := json.Unmarshal(body, &statuses)
	if err != nil {
		return nil, err, http.StatusBadRequest
	}

	for i := range statuses {
		var param map[string]string
		param = make(map[string]string)
		param["certId"] = statuses[i].CertId
		cert, err, _ := CertificateOut(setup, &param)
		if err != nil {
			continue
		}

		if cert == nil {
			continue
		}
		certificate := cert.(Certificate)

		certificate.IsDeleted = statuses[i].IsDeleted
		certificate.IsOpen = statuses[i].IsOpen
		certificate.IsCompleted = statuses[i].IsCompleted

		nerCert, err := json.Marshal(certificate)

		_, err, _ = CertificateIn(setup, nerCert)
		if err != nil {
			continue
		}
		statuses[i].IsChangedOnChain = true
	}
	logger.Info("-----------------------------ChangeStatus END---------------------")
	return statuses, nil, http.StatusOK
}

func OpenListRichQuery(setup *fabricClient.FabricSetup, body []byte, isLogin bool) (interface{}, error, int) {

	logger.Info("-----------------------------OpenListRichQuery BEGIN---------------------")
	var queryString string

	queryString = "{\"selector\":{\"$and\":[{\"isOpen\":true},{\"isCompleted\": true},{\"isDeleted\": false}"

	var queryConditions QueryConditions
	err := json.Unmarshal(body, &queryConditions)
	if err != nil {
		return err.Error(), err, http.StatusBadRequest
	}

	queryConditions.IsOpen = true

	queryString = editQueryString(queryString, &queryConditions)

	logger.Info("-----------------------------OpenListRichQuery END---------------------")
	return CertificateRichQuery(setup, queryConditions, isLogin, queryString, &openListBookmarks)
}

func editQueryString(queryString string, queryConditions *QueryConditions) string {

	queryString += editEqString(queryConditions.CertType, "certType")
	queryString += editEqString(queryConditions.CertId, "certId")
	queryString += editEqString(queryConditions.EntrustOrg, "entrustOrg")
	queryString += editEqString(queryConditions.InstrumentName, "instrumentName")

	queryString += editCoString(queryConditions.StartCreateDate, "updateDate", "$gte")
	queryString += editCoString(queryConditions.EndCreateDate, "updateDate", "$lte")
	queryString += editCoString(queryConditions.StartCalibDate, "calibrateDate", "$gte")
	queryString += editCoString(queryConditions.EndCalibDate, "calibrateDate", "$lte")

	queryString += "]},\"sort\": [{\"updateDate\": \"desc\"},{\"certId\": \"desc\"}],\"use_index\": [\"indexSortDoc\",\"indexSort\"]}"

	return queryString
}

func editEqString(value string, str string) string {
	value = strings.Replace(value, " ", "", -1)
	value = strings.Replace(value, "\n", "", -1)
	queryStr := ""
	if value != "" {
		queryStr = fmt.Sprintf(",{\""+str+"\": \"%v\"}", value)
	}
	return queryStr
}

func editCoString(value string, str string, comStr string) string {
	value = strings.Replace(value, " ", "", -1)
	value = strings.Replace(value, "\n", "", -1)
	queryStr := ""
	if value != "" {
		queryStr = fmt.Sprintf(",{\""+str+"\": {\""+comStr+"\":\"%v\"}}", value)
	}
	return queryStr
}

func DeletedListRichQuery(setup *fabricClient.FabricSetup, body []byte, isLogin bool) (interface{}, error, int) {

	logger.Info("-----------------------------DeletedListRichQuery BEGIN---------------------")
	var queryString string
	queryString = "{\"selector\":{\"$and\":[{\"isDeleted\": true}"
	var queryConditions QueryConditions
	err := json.Unmarshal(body, &queryConditions)
	if err != nil {
		return err.Error(), err, http.StatusBadRequest
	}

	queryString = editQueryString(queryString, &queryConditions)

	logger.Info("-----------------------------DeletedListRichQuery END---------------------")
	return CertificateRichQuery(setup, queryConditions, isLogin, queryString, &deletedListBookmarks)
}

func DraftListRichQuery(setup *fabricClient.FabricSetup, body []byte, isLogin bool) (interface{}, error, int) {

	logger.Info("-----------------------------DraftListRichQuery BEGIN---------------------")
	var queryString string
	queryString = "{\"selector\":{\"$and\":[{\"isOpen\":false},{\"isCompleted\": false},{\"isDeleted\": false}"
	var queryConditions QueryConditions
	err := json.Unmarshal(body, &queryConditions)
	if err != nil {
		return err.Error(), err, http.StatusBadRequest
	}

	queryString = editQueryString(queryString, &queryConditions)

	logger.Info("-----------------------------DraftListRichQuery END---------------------")
	return CertificateRichQuery(setup, queryConditions, isLogin, queryString, &draftListBookmarks)
}

func CertificateRichQuery(setup *fabricClient.FabricSetup, queryConditions QueryConditions, isLogin bool, queryString string, bookmarks *[]string) (interface{}, error, int) {

	logger.Info("-----------------------------CertificateRichQuery BEGIN---------------------")
	logger.Error(queryString + "|||||||||||||||")

	if queryConditions.IsOpen == false && !isLogin {
		return nil, errors.New("Should login"), http.StatusUnauthorized
	} else {
		pageSize := queryConditions.PageSize
		pageIndex := queryConditions.PageIndex

		if pageIndex < 1 {
			return nil, errors.New("PageIndex value should >=1"), http.StatusBadRequest
		}

		var bookmark string
		if pageIndex == 1 {
			bookmark = ""
		} else if pageIndex > 1 {
			//当前请求的页数对应没有存储书签，需要依次请求1到pageIndex的书签
			//循环遍历书签列表，找到最大位的有值书签
			for index := 1; index < pageIndex; index++ {
				if index < len(*bookmarks) {
					if (*bookmarks)[index] == "" || &(*bookmarks)[index] == nil {
						bookmark = (*bookmarks)[index-1]
						//调用获取方法，循环更新bookmark
						newlist, err, code := getNewBookmarks(setup, queryString, pageSize, bookmark)
						if err != nil {
							return err.Error(), err, code
						}
						(*bookmarks)[index] = newlist.Bookmark
					} else {
						continue
					}
				} else {
					bookmark = (*bookmarks)[index-1]
					//调用获取方法，循环更新bookmark
					newlist, err, code := getNewBookmarks(setup, queryString, pageSize, bookmark)
					if err != nil {
						return err.Error(), err, code
					}
					(*bookmarks)[index] = newlist.Bookmark
				}
			}
		}
		bookmark = (*bookmarks)[pageIndex-1]
		//调用获取方法，循环更新bookmark
		listInter, err, code := getNewBookmarks(setup, queryString, pageSize, bookmark)
		if err != nil {
			return err.Error(), err, code
		}

		if len(*bookmarks) > pageIndex {
			if listInter.Bookmark != (*bookmarks)[pageIndex] {
				(*bookmarks)[pageIndex] = listInter.Bookmark
				//书签过时，后面的全部置空
				for i := pageIndex + 1; i < len(*bookmarks); i++ {
					(*bookmarks)[i] = ""
				}
			}
		} else if len(*bookmarks) == pageIndex {
			*bookmarks = append(*bookmarks, listInter.Bookmark)
		}

		var list List
		list.PageCount = listInter.PageCount
		list.PageIndex = pageIndex
		list.Certs = listInter.Certs
		return list, nil, http.StatusOK
	}
}

func getNewBookmarks(setup *fabricClient.FabricSetup, queryString string, pageSize int, bookmark string) (*ListInternal, error, int) {
	args := []string{queryString, strconv.Itoa(pageSize), bookmark}

	response, err := setup.Query("queryList", args)
	if err != nil {
		return nil, err, http.StatusBadRequest
	}

	if response == "" {
		return nil, nil, http.StatusNotFound
	}

	var listInter ListInternal
	err = json.Unmarshal([]byte(response), &listInter)
	if err != nil {
		return nil, err, http.StatusNotImplemented
	}
	return &listInter, nil, http.StatusOK
}

func deleteFileOnDisk(localPath string) {
	logger.Debugf("remove file: %s", localPath)
	if err := os.Remove(localPath); err != nil {
		logger.Error(err)
	}
	dirsList := make([]string, 0, 0)
	for dir := path.Dir(localPath); dir != "./tempUploadFiles" && len(dir) > len("./tempUploadFiles"); dir = path.Dir(dir) {
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
