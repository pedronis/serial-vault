// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2017-2018 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package manage

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/CanonicalLtd/serial-vault/crypt"
	"github.com/CanonicalLtd/serial-vault/service"
	"github.com/snapcore/snapd/asserts"
)

// ClientCommand is the command for the serial-vault test client
type ClientCommand struct {
	Brand        string `short:"b" long:"brand" description:"The brand-id of the device" required:"yes"`
	Model        string `short:"m" long:"model" description:"The model name of the device" required:"yes"`
	SerialNumber string `short:"s" long:"serial" description:"The serial number of the device" required:"yes"`
	URL          string `short:"u" long:"url" description:"The base URL of the serial vault API" required:"yes"`
	APIKey       string `short:"a" long:"api" description:"The API Key for the serial vault" required:"yes"`
}

// Execute the database schema updates
func (cmd ClientCommand) Execute(args []string) error {
	// Create a serial-request assertion
	serialRequest, err := cmd.generateSerialRequestAssertion()
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}

	// Send it to the serial vault via HTTPS
	serialAssertion, err := cmd.getSerial(serialRequest)
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}

	fmt.Println(serialAssertion)
	return nil
}

func (cmd ClientCommand) generatePrivateKey() (asserts.PrivateKey, error) {
	signingKey, err := ioutil.ReadFile("./keystore/TestDeviceKey.asc")
	if err != nil {
		return nil, err
	}
	encodedSigningKey := base64.StdEncoding.EncodeToString(signingKey)

	privateKey, _, err := crypt.DeserializePrivateKey(encodedSigningKey)
	return privateKey, nil
}

func (cmd ClientCommand) generateSerialRequestAssertion() (string, error) {
	privateKey, err := cmd.generatePrivateKey()
	if err != nil {
		return "", err
	}
	encodedPubKey, err := asserts.EncodePublicKey(privateKey.PublicKey())
	if err != nil {
		return "", err
	}

	// Generate a request-id
	r, _ := cmd.getRequestID()

	headers := map[string]interface{}{
		"brand-id":   cmd.Brand,
		"device-key": string(encodedPubKey),
		"request-id": r,
		"model":      cmd.Model,
		"serial":     cmd.SerialNumber,
	}

	sreq, err := asserts.SignWithoutAuthority(asserts.SerialRequestType, headers, []byte(""), privateKey)
	if err != nil {
		return "", err
	}

	assertSR := asserts.Encode(sreq)
	return string(assertSR), nil
}

func (cmd ClientCommand) getRequestID() (string, error) {
	// Format the URL and headers for the HTTP call
	req := cmd.getHTTPRequest("request-id", "")

	// Call the /request-id API
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error fetching the request-id")
		return "", err
	}
	defer resp.Body.Close()

	// Parse the API response
	result := service.RequestIDResponse{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		log.Println("Error parsing the request-id")
		return "", err
	}

	return result.RequestID, nil
}

func (cmd ClientCommand) getSerial(serialRequest string) (string, error) {
	// Format the URL and headers for the HTTP call
	req := cmd.getHTTPRequest("serial", serialRequest)

	// Call the /request-id API
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error fetching the serial assertion")
		return "", err
	}
	defer resp.Body.Close()

	// Check the content-type to see if we have a JSON error response
	if resp.Header.Get("Content-Type") == "application/json; charset=UTF-8" {
		// Parse the API response
		result := service.SignResponse{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			log.Println("Error parsing the serial assertion error")
			return "", err
		}
		message := fmt.Sprintf("%s: %s", result.ErrorCode, result.ErrorMessage)
		return "", errors.New(message)
	}

	// Must have a valid assertion
	body, err := ioutil.ReadAll(resp.Body)

	return string(body), err
}

func (cmd ClientCommand) getHTTPRequest(method, body string) *http.Request {
	// Format the URL and headers for the HTTP call
	url := fmt.Sprintf("%s%s", cmd.URL, method)
	req, _ := http.NewRequest("POST", url, bytes.NewBufferString(body))
	req.Header.Set("api-key", cmd.APIKey)
	req.Header.Set("Content-Type", "application/json")

	return req
}
