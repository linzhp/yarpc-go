// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package yarpctest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/transport/grpc"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/yarpc/x/yarpctest/api"
)

// HTTPRequest creates a new YARPC http request.
func HTTPRequest(options ...api.RequestOption) api.Action {
	opts := api.NewRequestOpts()
	for _, option := range options {
		option.ApplyRequest(&opts)
	}
	return api.ActionFunc(func(t api.TestingT) {
		trans := http.NewTransport()
		out := trans.NewSingleOutbound(fmt.Sprintf("http://127.0.0.1:%d/", opts.Port))

		require.NoError(t, trans.Start())
		defer func() { assert.NoError(t, trans.Stop()) }()

		require.NoError(t, out.Start())
		defer func() { assert.NoError(t, out.Stop()) }()

		resp, cancel, err := sendRequest(out, opts.GiveRequest)
		defer cancel()
		validateError(t, err, opts.ExpectedError)
		if opts.ExpectedError == nil {
			validateResponse(t, resp, opts.ExpectedResponse)
		}
	})
}

// TChannelRequest creates a new tchannel request.
func TChannelRequest(options ...api.RequestOption) api.Action {
	opts := api.NewRequestOpts()
	for _, option := range options {
		option.ApplyRequest(&opts)
	}
	return api.ActionFunc(func(t api.TestingT) {
		trans, err := tchannel.NewTransport(tchannel.ServiceName(opts.GiveRequest.Caller))
		require.NoError(t, err)
		out := trans.NewSingleOutbound(fmt.Sprintf("127.0.0.1:%d", opts.Port))

		require.NoError(t, trans.Start())
		defer func() { assert.NoError(t, trans.Stop()) }()

		require.NoError(t, out.Start())
		defer func() { assert.NoError(t, out.Stop()) }()

		resp, cancel, err := sendRequest(out, opts.GiveRequest)
		defer cancel()
		validateError(t, err, opts.ExpectedError)
		if opts.ExpectedError == nil {
			validateResponse(t, resp, opts.ExpectedResponse)
		}
	})
}

// GRPCRequest creates a new grpc unary request.
func GRPCRequest(options ...api.RequestOption) api.Action {
	opts := api.NewRequestOpts()
	for _, option := range options {
		option.ApplyRequest(&opts)
	}
	return api.ActionFunc(func(t api.TestingT) {
		trans := grpc.NewTransport()
		out := trans.NewSingleOutbound(fmt.Sprintf("127.0.0.1:%d", opts.Port))

		require.NoError(t, trans.Start())
		defer func() { assert.NoError(t, trans.Stop()) }()

		require.NoError(t, out.Start())
		defer func() { assert.NoError(t, out.Stop()) }()

		resp, cancel, err := sendRequest(out, opts.GiveRequest)
		defer cancel()
		validateError(t, err, opts.ExpectedError)
		if opts.ExpectedError == nil {
			validateResponse(t, resp, opts.ExpectedResponse)
		}
	})
}

func sendRequest(out transport.UnaryOutbound, request *transport.Request) (*transport.Response, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	resp, err := out.Call(ctx, request)
	return resp, cancel, err
}

func validateError(t api.TestingT, actualErr error, wantError error) {
	if wantError != nil {
		require.Error(t, actualErr)
		require.Contains(t, actualErr.Error(), wantError.Error())
		return
	}
	require.NoError(t, actualErr)
}

func validateResponse(t api.TestingT, actualResp *transport.Response, expectedResp *transport.Response) {
	var actualBody []byte
	var expectedBody []byte
	var err error
	if actualResp.Body != nil {
		actualBody, err = ioutil.ReadAll(actualResp.Body)
		require.NoError(t, err)
	}
	if expectedResp.Body != nil {
		expectedBody, err = ioutil.ReadAll(expectedResp.Body)
		require.NoError(t, err)
	}
	assert.Equal(t, string(actualBody), string(expectedBody))
}

// UNARY-SPECIFIC REQUEST OPTIONS

// Body sets the body on a request to the raw representation of the msg field.
func Body(msg string) api.RequestOption {
	return api.RequestOptionFunc(func(opts *api.RequestOpts) {
		opts.GiveRequest.Body = bytes.NewBufferString(msg)
	})
}

// ExpectError creates an assertion on the request response to validate the
// error.
func ExpectError(errMsg string) api.RequestOption {
	return api.RequestOptionFunc(func(opts *api.RequestOpts) {
		opts.ExpectedError = errors.New(errMsg)
	})
}

// ExpectRespBody will assert that the response body matches at the end of the
// request.
func ExpectRespBody(body string) api.RequestOption {
	return api.RequestOptionFunc(func(opts *api.RequestOpts) {
		opts.ExpectedResponse.Body = ioutil.NopCloser(bytes.NewBufferString(body))
	})
}

// GiveAndExpectLargeBodyIsEchoed creates an extremely large random byte buffer
// and validates that the body is echoed back to the response.
func GiveAndExpectLargeBodyIsEchoed(numOfBytes int) api.RequestOption {
	return api.RequestOptionFunc(func(opts *api.RequestOpts) {
		body := bytes.Repeat([]byte("t"), numOfBytes)
		opts.GiveRequest.Body = bytes.NewReader(body)
		opts.ExpectedResponse.Body = ioutil.NopCloser(bytes.NewReader(body))
	})
}