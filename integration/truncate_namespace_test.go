// +build integration

// Copyright (c) 2016 Uber Technologies, Inc.
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

package integration

import (
	"testing"
	"time"

	"github.com/m3db/m3db/generated/thrift/rpc"
	"github.com/m3db/m3db/integration/generate"
	"github.com/m3db/m3db/ts"
	xtime "github.com/m3db/m3x/time"

	"github.com/stretchr/testify/require"
)

func TestTruncateNamespace(t *testing.T) {
	if testing.Short() {
		t.SkipNow() // Just skip if we're doing a short run
	}
	// Test setup
	testOpts := newTestOptions(t)
	testSetup, err := newTestSetup(t, testOpts)
	require.NoError(t, err)
	defer testSetup.close()

	blockSize := testOpts.CommitLogBlockSize()

	// Start the server
	log := testSetup.storageOpts.InstrumentOptions().Logger()
	log.Debug("truncate namespace test")
	require.NoError(t, testSetup.startServer())
	log.Debug("server is now up")

	// Stop the server
	defer func() {
		require.NoError(t, testSetup.stopServer())
		log.Debug("server is now down")
	}()

	// Write test data
	now := testSetup.getNowFn()
	seriesMaps := make(map[time.Time]generate.SeriesBlock)
	inputData := []struct {
		namespace ts.ID
		conf      generate.BlockConfig
	}{
		{testNamespaces[0], generate.BlockConfig{[]string{"foo"}, 100, now}},
		{testNamespaces[1], generate.BlockConfig{[]string{"bar"}, 50, now.Add(blockSize)}},
	}
	for _, input := range inputData {
		testSetup.setNowFn(input.conf.Start)
		testData := generate.Block(input.conf)
		seriesMaps[input.conf.Start] = testData
		require.NoError(t, testSetup.writeBatch(input.namespace, testData))
	}
	log.Debug("test data is now written")

	fetchReq := rpc.NewFetchRequest()
	fetchReq.ID = "foo"
	fetchReq.NameSpace = testNamespaces[1].String()
	fetchReq.RangeStart = xtime.ToNormalizedTime(now, time.Second)
	fetchReq.RangeEnd = xtime.ToNormalizedTime(now.Add(blockSize), time.Second)
	fetchReq.ResultTimeType = rpc.TimeType_UNIX_SECONDS

	log.Debug("fetching data from nonexistent namespace")
	fetchReq.NameSpace = "nonexistent"
	_, err = testSetup.fetch(fetchReq)
	require.Error(t, err)

	log.Debug("fetching data from wrong namespace")
	fetchReq.NameSpace = testNamespaces[1].String()
	res, err := testSetup.fetch(fetchReq)
	require.NoError(t, err)
	require.Equal(t, 0, len(res))

	log.Debugf("fetching data from namespace %s", testNamespaces[0])
	fetchReq.NameSpace = testNamespaces[0].String()
	res, err = testSetup.fetch(fetchReq)
	require.NoError(t, err)
	require.Equal(t, 100, len(res))

	log.Debugf("truncate namespace %s", testNamespaces[0])
	truncateReq := rpc.NewTruncateRequest()
	truncateReq.NameSpace = testNamespaces[0].Data().Get()
	truncated, err := testSetup.truncate(truncateReq)
	require.NoError(t, err)
	require.Equal(t, int64(1), truncated)

	log.Debugf("fetching data from namespace %s again", testNamespaces[0])
	res, err = testSetup.fetch(fetchReq)
	require.NoError(t, err)
	require.Equal(t, 0, len(res))

	log.Debugf("fetching data from a different namespace %s", testNamespaces[1])
	fetchReq.ID = "bar"
	fetchReq.NameSpace = testNamespaces[1].String()
	fetchReq.RangeStart = xtime.ToNormalizedTime(now.Add(blockSize), time.Second)
	fetchReq.RangeEnd = xtime.ToNormalizedTime(now.Add(blockSize*2), time.Second)
	res, err = testSetup.fetch(fetchReq)
	require.NoError(t, err)
	require.Equal(t, 50, len(res))
}
