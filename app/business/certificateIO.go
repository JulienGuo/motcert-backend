package business

import (
	"encoding/json"
	"errors"
	"github.com/op/go-logging"
	"gitlab.chainnova.com/motcert-backend/app/fabricClient"
	"net/http"
)

var logger = logging.MustGetLogger("Motcert.business")

var openListBookmarks []string
var closedListBookmarks []string
var unfinishedListBookmars []string

func init() {
	//可以加定时任务优化
	openListBookmarks = append(openListBookmarks, "")
	closedListBookmarks = append(closedListBookmarks, "")
	unfinishedListBookmars = append(unfinishedListBookmars, "")
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
	CreateDate          string `protobuf:"bytes,16,opt,name=createDate" json:"createDate"`                   //创建日期
}

type Status struct {
	CertId           string `json:"certId"`
	IsOpen           bool   `json:"isOpen"`
	IsChangedOnChain bool   `json:"isChangedOnChain"`
}

type QueryConditions struct {
	PageSize        int    `protobuf:"bytes,1,req,name=pageSize" json:"pageSize"`               //每页数据量
	PageIndex       int    `protobuf:"bytes,2,req,name=pageIndex" json:"pageIndex"`             //本次请求的页号
	IsOpen          bool   `protobuf:"bytes,3,req,name=isOpen" json:"isOpen"`                   //是否公开
	IsCompleted     bool   `protobuf:"bytes,4,req,name=isCompleted" json:"isCompleted"`         //是否完成
	CertType        string `protobuf:"bytes,5,opt,name=certType" json:"certType"`               //证书类型
	CertId          string `protobuf:"bytes,6,opt,name=certId" json:"certId"`                   //证书编号
	EntrustOrg      string `protobuf:"bytes,7,opt,name=entrustOrg" json:"entrustOrg"`           //委托单位
	InstrumentName  string `protobuf:"bytes,8,opt,name=instrumentName" json:"instrumentName"`   //器具名称
	StartCreateDate string `protobuf:"bytes,9,opt,name=startCreateDate" json:"startCreateDate"` //起始录入日期
	EndCreateDate   string `protobuf:"bytes,10,opt,name=endCreateDate" json:"endCreateDate"`    //结束录入日期
	StartCalibDate  string `protobuf:"bytes,11,opt,name=startCalibDate" json:"startCalibDate"`  //起始校准日期
	EndCalibDate    string `protobuf:"bytes,12,opt,name=endCalibDate" json:"endCalibDate"`      //结束校准日期
}

type ListInternal struct {
	PageCount int
	Bookmark  string
	Certs     []Certificate
}
type List struct {
	PageCount int
	PageIndex int
	Certs     []Certificate
}

func CertificateIn(setup *fabricClient.FabricSetup, body []byte) (interface{}, error, int) {

	logger.Info("-----CertificateIn-----")
	var certificate Certificate
	err := json.Unmarshal(body, &certificate)
	if err != nil {
		return nil, err, http.StatusBadRequest
	}

	logger.Infof("postCertificate: '%s'.\n", certificate)
	args := []string{string(body)}

	eventID := "postCertificateEvent"
	data, err := setup.Execute(eventID, "postCertificate", args)
	if err != nil {
		return nil, err, http.StatusNotImplemented
	}
	return data, nil, http.StatusOK
}

func CertificateOut(setup *fabricClient.FabricSetup, param map[string]string) (interface{}, error, int) {

	logger.Info("-----------------------------CertificateOut BEGIN---------------------")
	certId := param["certId"]
	args := []string{certId}
	var paraArgs []string
	for _, arg := range args {
		paraArgs = append(paraArgs, arg)
	}

	response, err := setup.Query("getCertificate", paraArgs)

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
	return certificate, nil, http.StatusOK
}

func ChangeStatus(setup *fabricClient.FabricSetup, body []byte) (interface{}, error, int) {
	var statuses []Status
	err := json.Unmarshal(body, &statuses)
	if err != nil {
		return nil, err, http.StatusBadRequest
	}

	for i, _ := range statuses {
		var param map[string]string
		param = make(map[string]string)
		param["certId"] = statuses[i].CertId
		cert, err, _ := CertificateOut(setup, param)
		if err != nil {
			continue
		}

		cert.(Certificate).IsOpen = statuses[i].IsOpen

		nerCert, err := json.Marshal(cert)

		_, err, _ = CertificateIn(setup, nerCert)
		if err != nil {
			continue
		}
		statuses[i].IsChangedOnChain = true
	}
	return statuses, nil, http.StatusOK
}

func CertificateRichQuery(setup *fabricClient.FabricSetup, body []byte, isLogin bool) (interface{}, error, int) {

	logger.Info("-----------------------------CertificateOut BEGIN---------------------")

	var queryConditions QueryConditions
	err := json.Unmarshal(body, &queryConditions)
	if err != nil {
		return err.Error(), err, http.StatusBadRequest
	}

	if queryConditions.IsOpen == false && !isLogin {
		return nil, errors.New("Should login"), http.StatusUnauthorized
	} else {
		var queryString string
		if queryConditions.IsOpen {
			queryString = "{\"selector\":{\"isOpen\":true}}"
		}
		pageSize := queryConditions.PageSize
		pageIndex := queryConditions.PageIndex

		var bookmark string
		if pageIndex == 1 {
			bookmark = ""
		} else if pageIndex > 1 {
			for index, mark := range openListBookmarks {
				if index > 0 && index < pageIndex {

				} else if index == pageIndex {
					if mark == "" || &mark == nil {

					}
				}
			}
		}

		args := []string{queryString, string(pageSize), bookmark}

		var paraArgs []string
		for _, arg := range args {
			paraArgs = append(paraArgs, arg)
		}

		response, err := setup.Query("queryList", paraArgs)
		if err != nil {
			return nil, err, http.StatusBadRequest
		}

		if response == "" {
			return nil, err, http.StatusNotFound
		}

		var listInter ListInternal
		err = json.Unmarshal([]byte(response), &listInter)
		if err != nil {
			return nil, err, http.StatusNotImplemented
		}
		if len(openListBookmarks) >= pageIndex {
			if listInter.Bookmark != openListBookmarks[pageIndex] {
				openListBookmarks[pageIndex] = listInter.Bookmark
				//书签过时
				////用go routine去更新书签
				//go
				for i := pageIndex + 1; i < listInter.PageCount; i++ {
					openListBookmarks[i] = ""
				}
			}
		} else if len(openListBookmarks) < pageIndex+1 {
			//书签没有初始化
			////用go routine去更新书签
			//go
		}

		var list List
		list.PageCount = listInter.PageCount
		list.PageIndex = queryConditions.PageIndex
		list.Certs = listInter.Certs
		return list, nil, http.StatusOK

	}

}
