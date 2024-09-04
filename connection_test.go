package network

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mateothegreat/go-multilog/multilog"
	"github.com/stretchr/testify/suite"
)

type SimpleStruct struct {
	Foo string `binary:"[24]string"`
}

func (s SimpleStruct) Encode() ([]byte, error) {
	return json.Marshal(s)
}

func (s SimpleStruct) Decode(data []byte) error {
	return json.Unmarshal(data, &s)
}

type TestSourceSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc
	server *Connection[SimpleStruct]
	client *Connection[SimpleStruct]
}

func TestSuiteTest(t *testing.T) {
	suite.Run(t, new(TestSourceSuite))
}

func (suite *TestSourceSuite) SetupSuite() {
	multilog.RegisterLogger(multilog.LogMethod("console"), multilog.NewConsoleLogger(&multilog.NewConsoleLoggerArgs{
		Level:  multilog.DEBUG,
		Format: multilog.FormatText,
	}))
	suite.ctx, suite.cancel = context.WithCancel(context.Background())
}

func (suite *TestSourceSuite) TearDownSuite() {
	suite.cancel()
}

func (suite *TestSourceSuite) TestConnection() {
	suite.server = NewConnection[SimpleStruct]("receiver", "127.0.0.1:5200", 500*time.Millisecond)
	if err := suite.server.Listen(); err != nil {
		suite.FailNow("Failed to listen", err)
	}

	time.Sleep(1000 * time.Millisecond)

	suite.client = NewConnection[SimpleStruct]("sender", "127.0.0.1:5200", 500*time.Millisecond)
	if err := suite.client.Connect(); err != nil {
		suite.FailNow("Failed to connect", err)
	}

	time.Sleep(1000 * time.Millisecond)
	n, err := suite.client.Write(&SimpleStruct{Foo: "bar"})
	suite.NoError(err)
	suite.Equal(13, n)
	time.Sleep(1000 * time.Millisecond)
	suite.server.Close()

	// time.Sleep(100 * time.Millisecond)

	// suite.Equal(Disconnected, suite.client.Status)
}
