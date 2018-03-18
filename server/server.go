package server

import (
	"encoding/json"
	"encoding/base64"
	"fmt"
	"net/http"
	"github.com/blk-io/crux/enclave"
	"github.com/kevinburke/nacl"
	"github.com/blk-io/crux/api"
	"encoding/hex"
)

type TransactionManager struct {
	Key nacl.Key
	Enclave enclave.Enclave
}

func (s *TransactionManager) Upcheck(w http.ResponseWriter, req *http.Request) {
	fmt.Fprint(w, "I'm up!")
}

func (s *TransactionManager) Send(w http.ResponseWriter, req *http.Request) {
	var sendReq api.SendRequest
	if err := json.NewDecoder(req.Body).Decode(&sendReq); err != nil {
		invalidBody(w, req, err)
	} else {
		payload, err := base64.StdEncoding.DecodeString(sendReq.Payload)
		if err != nil {
			decodeError(w, req, "payload", sendReq.Payload, err)
			return
		}
		decoded, err := base64.StdEncoding.DecodeString(sendReq.From)
		if err != nil {
			decodeError(w, req, "sender", sendReq.From, err)
			return
		}
		sender := hex.EncodeToString(decoded)

		recipients := make([]string, len(sendReq.To))
		for _, value := range sendReq.To {
			recipient, err := base64.StdEncoding.DecodeString(sendReq.From)
			if err != nil {
				decodeError(w, req, "recipient", value, err)
				return
			} else {
				recipients = append(recipients, hex.EncodeToString(recipient))
			}
		}

		key, err := s.Enclave.Store(&payload, sender, recipients)
		if err != nil {
			badRequest(w,
				fmt.Sprintf("Unable to store key: %s, with payload: %s, error: %s\n",
					key, payload, err))
		} else {
			encodedKey := base64.StdEncoding.EncodeToString(key)
			sendResp := api.SendResponse{Key : encodedKey}
			json.NewEncoder(w).Encode(sendResp)
			w.Header().Set("Content-Type", "application/json")
		}
	}
}

func (s *TransactionManager) Receive(w http.ResponseWriter, req *http.Request) {
	var receiveReq api.ReceiveRequest
	if err := json.NewDecoder(req.Body).Decode(&receiveReq); err != nil {
		invalidBody(w, req, err)
	} else {
		key, err := base64.StdEncoding.DecodeString(receiveReq.Key)
		if err != nil {
			decodeError(w, req, "key", receiveReq.Key, err)
		} else {
			var payload []byte
			payload, err = s.Enclave.Retrieve(s.Key, &key)
			if err != nil {
				badRequest(w,
					fmt.Sprintf("Unable to retrieve payload for key: %s, error: %s\n",
						receiveReq.Key, err))
			} else {
				encodedPayload := base64.StdEncoding.EncodeToString(payload)
				sendResp := api.ReceiveResponse{Payload: encodedPayload}
				json.NewEncoder(w).Encode(sendResp)
				w.Header().Set("Content-Type", "application/json")
			}
		}
	}
}

func (s *TransactionManager) Delete(w http.ResponseWriter, req *http.Request) {
	var deleteReq api.DeleteRequest
	if err := json.NewDecoder(req.Body).Decode(&deleteReq); err != nil {
		invalidBody(w, req, err)
	} else {
		key, err := base64.StdEncoding.DecodeString(deleteReq.Key)
		if err != nil {
			decodeError(w, req, "key", deleteReq.Key, err)
		} else {
			err = s.Enclave.Delete(&key)
			if err != nil {
				badRequest(w, fmt.Sprintf("Unable to delete key: %s, error: %s\n", key, err))
			}
		}
	}
}

func (s *TransactionManager) Push(w http.ResponseWriter, req *http.Request) {
	payload := make([]byte, 256)
	req.Body.Read(payload)

	digestHash, err := s.Enclave.StorePayload(payload)
	if err != nil {
		badRequest(w, fmt.Sprintf("Unable to store payload, error: %s\n", err))
	} else {
		encodedDigestHash := base64.StdEncoding.EncodeToString(digestHash)
		fmt.Fprint(w, "%s", encodedDigestHash)
	}
}


func invalidBody(w http.ResponseWriter, req *http.Request, err error) {
	req.Body.Close()
	badRequest(w, fmt.Sprintf("Invalid request: %s, error: %s\n", req.URL, err))
}

func decodeError(w http.ResponseWriter, req *http.Request, name string, value string, err error) {
	badRequest(w,
		fmt.Sprintf("Invalid request: %s, unable to decode %s: %s, error: %s\n",
			req.URL, name, value, err))
}

func badRequest(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(w, message)
}