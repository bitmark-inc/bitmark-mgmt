package api

import (
	"encoding/json"
	"github.com/bitmark-inc/bitmark-webgui/fault"
	"github.com/bitmark-inc/bitmark-webgui/services"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/logger"
	"net/http"
)

type BitmarkCliIssueResponse struct {
	AssetId        string               `json:"assetId"`
	IssueIds       []string             `json:"issueIds"`
	PaymentAddress []block.MinerAddress `json:"paymentAddress"`
}

type BitmarkCliTransferResponse struct {
	TransferId     string               `json: transferId`
	PaymentAddress []block.MinerAddress `json:"paymentAddress"`
}

//POST /api/bitmarkCli/*
func BitmarkCliExec(w http.ResponseWriter, req *http.Request, log *logger.L, command string) {
	log.Infof("POST /api/bitmarCli/%s", command)
	response := &Response{
		Ok:     false,
		Result: nil,
	}

	switch command {
	case "setup":
		var request services.BitmarkCliSetupType
		decoder := json.NewDecoder(req.Body)
		err := decoder.Decode(&request)
		if nil != err {
			log.Errorf("Error: %v", err)
			response.Result = "bitmark-cli request parsing error"
			if err := writeApiResponseAndSetCookie(w, response); nil != err {
				log.Errorf("Error: %v", err)
			}
			return
		}

		_, err = bitmarkCliService.Setup(request)
		if nil != err {
			response.Result = "bitmark-cli setup error"
		} else {
			response.Ok = true
			response.Result = "Success"
		}
	case "issue":
		var request services.BitmarkCliIssueType
		decoder := json.NewDecoder(req.Body)
		err := decoder.Decode(&request)
		if nil != err {
			log.Errorf("Error: %v", err)
			response.Result = "bitmark-cli request parsing error"
			if err := writeApiResponseAndSetCookie(w, response); nil != err {
				log.Errorf("Error: %v", err)
			}
			return
		}

		output, err := bitmarkCliService.Issue(request)
		if nil != err {
			response.Result = "bitmark-cli issue error"
		} else {
			var jsonIssue BitmarkCliIssueResponse
			if err := json.Unmarshal(output, &jsonIssue); nil != err {
				log.Errorf("Error: %v", err)
			} else {
				response.Ok = true
				response.Result = jsonIssue
			}
		}
	case "transfer":
		var request services.BitmarkCliTransferType
		decoder := json.NewDecoder(req.Body)
		err := decoder.Decode(&request)
		if nil != err {
			log.Errorf("Error: %v", err)
			response.Result = "bitmark-cli request parsing error"
			if err := writeApiResponseAndSetCookie(w, response); nil != err {
				log.Errorf("Error: %v", err)
			}
			return
		}

		output, err := bitmarkCliService.Transfer(request)
		if nil != err {
			response.Result = "bitmark-cli transfer error"
		} else {
			var jsonTransfer BitmarkCliTransferResponse
			if err := json.Unmarshal(output, &jsonTransfer); nil != err {
				log.Errorf("Error: %v", err)
			} else {
				response.Ok = true
				response.Result = jsonTransfer
			}
		}
	default:
		response.Result = fault.ErrInvalidCommandType
	}

	if err := writeApiResponseAndSetCookie(w, response); nil != err {
		log.Errorf("Error: %v", err)
	}
}

// func parseRequest(req *http.Request, command string) (interface{}, error) {
// 	decoder := json.NewDecoder(req.Body)
// 	var request interface{}
// 	switch command {
// 	case "setup":
// 		request = services.BitmarkCliSetupType{}
// 	case "issue":
// 		request = services.BitmarkCliIssueType{}
// 	case "transfer":
// 		request = services.BitmarkCliTransferType{}
// 	default:
// 		return nil, fault.ErrInvalidCommandType
// 	}

// 	err := decoder.Decode(&request)
// 	if nil != err {
// 		return nil, err
// 	}

// 	return request, nil
// }