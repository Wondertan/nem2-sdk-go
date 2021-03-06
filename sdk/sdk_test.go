// Copyright 2018 ProximaX Limited. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package sdk

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	address = "http://10.32.150.136:3000"
)

func setupWithAddress(adr string) *Client {
	conf, err := NewConfig(adr, TestNet)
	if err != nil {
		panic(err)
	}

	return NewClient(nil, conf)
}

func setup() (*Client, string) {
	conf, err := NewConfig(address, TestNet)
	if err != nil {
		panic(err)
	}

	return NewClient(nil, conf), address
}

// Create a mock server
func setupMockServer() (client *Client, mux *http.ServeMux, serverURL string, teardown func(), err error) {
	// individual tests will provide API mock responses
	mux = http.NewServeMux()

	server := httptest.NewServer(mux)

	conf, err := NewConfig(server.URL, TestNet)
	if err != nil {
		return nil, nil, "", nil, err
	}

	client = NewClient(nil, conf)

	return client, mux, server.URL, server.Close, nil
}

// Bool is a helper routine that allocates a new bool value
// to store v and returns a pointer to it.
func Bool(v bool) *bool { return &v }

// Int is a helper routine that allocates a new int value
// to store v and returns a pointer to it.
func Int(v int) *int { return &v }

// Int64 is a helper routine that allocates a new int64 value
// to store v and returns a pointer to it.
func Int64(v int64) *int64 { return &v }

// Uint64 is a helper routine that allocates a new int64 value
// to store v and returns a pointer to it.
func Uint64(v uint64) *uint64 { return &v }

// String is a helper routine that allocates a new string value
// to store v and returns a pointer to it.
func String(v string) *string { return &v }

type sParam struct {
	desc     string
	req      bool
	Type     string
	defValue interface{}
}

type sRouting struct {
	resp   string
	params map[string]sParam
}

func (r *sRouting) checkParams(req *http.Request) (badParams []string, err error) {
	for key, val := range r.params {

		if key == "body" {
			b, err := ioutil.ReadAll(req.Body)
			if err != nil {
				err = errors.New("failed during reading Body")
				return badParams, err
			}
			if (len(b) == 0) || (bytes.Contains(b, []byte("null"))) {
				badParams = append(badParams, "body is empty")
			}
			//	todo: add check struct to match the request requirements
		} else if valueParam := req.FormValue(key); (val.req) && (valueParam == "") {
			badParams = append(badParams, key)
		} else if val.Type > "" {
			//	check type is later
			if valueParam != val.Type {
				err = errors.New("bad type param")
				return
			}
		}
	}

	return
}

type mockService struct {
	*Client
	mux  *http.ServeMux
	lock sync.Locker
}

func NewMockServerWithRouters(routers map[string]sRouting) *mockService {

	serv := NewMockServer()

	serv.addRouters(routers)

	return serv
}

func NewMockServer() *mockService {
	client, mux, _, teardown, err := setupMockServer()

	if err != nil {
		panic(err)
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		//	mock router as default
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "%s not found in mock routers", r.URL)
		fmt.Println(r.URL)
	})
	time.AfterFunc(time.Minute*5, teardown)

	return &mockService{mux: mux, Client: client}
}

func (serv *mockService) addRouters(routers map[string]sRouting) {
	for path, route := range routers {
		apiRoute := route
		serv.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			// Mock JSON response
			if params, err := apiRoute.checkParams(r); (len(params) > 0) || (err != nil) {
				w.WriteHeader(http.StatusBadRequest)
				if len(params) > 0 {
					p := strings.Join(params, ",")
					fmt.Fprintf(w, "bad params - %s", p)
				}
				if err != nil {
					fmt.Fprint(w, "error during params validate - ", err)
				}
			} else {
				w.Write([]byte(apiRoute.resp))
			}
		})

	}

}

var (
	ctx           = context.TODO()
	routeNeedBody = map[string]sParam{"body": {desc: "required body"}}
)

func validateResp(resp *http.Response, t *testing.T) bool {
	if !assert.NotNil(t, resp) {
		return false
	}
	if !assert.Equal(t, 200, resp.StatusCode) {
		t.Logf("%#v", resp.Body)
		return false
	}
	return true
}

//using different numbers from original javs sdk because of signed and unsigned transformation
//ex. uint64(-8884663987180930485) = 9562080086528621131
func TestBigIntegerToHex_bigIntegerNEMAndXEMToHex(t *testing.T) {
	testBigInt(t, "9562080086528621131", "84b3552d375ffa4b")
	testBigInt(t, "15358872602548358953", "d525ad41d95fcf29")
}
func testBigInt(t *testing.T, str, hexStr string) {
	i, ok := (&big.Int{}).SetString(str, 10)
	assert.True(t, ok)
	result := BigIntegerToHex(i)
	assert.Equal(t, hexStr, result)

}
