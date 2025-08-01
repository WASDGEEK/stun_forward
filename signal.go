// signal.go
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type SignalData struct {
	Role string `json:"role"`
	Room string `json:"room"`
	Data string `json:"data"`
}

func PostSignal(url, role, room, data string) error {
	body, _ := json.Marshal(SignalData{Role: role, Room: room, Data: data})
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New("non-200 response from signal server")
	}
	return nil
}

func WaitForPeerData(url, peerRole, room string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("%s?role=%s&room=%s", url, peerRole, room))
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		if resp.StatusCode == 200 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				resp.Body.Close()
				time.Sleep(1 * time.Second)
				continue
			}
			resp.Body.Close()
			if len(body) > 0 {
				return string(body), nil
			}
		} else {
			resp.Body.Close()
		}

		time.Sleep(1 * time.Second)
	}
	return "", errors.New("timeout waiting for peer data")
}