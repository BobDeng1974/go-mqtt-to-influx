package converter

import (
	"github.com/golang/mock/gomock"
	"github.com/koestler/go-mqtt-to-influxdb/converter/mock"
	"github.com/koestler/go-mqtt-to-influxdb/influxDbClient"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

//go:generate mockgen -destination=mock/converter_mock.go -package converter_mock github.com/koestler/go-mqtt-to-influxdb/converter Config,Statistics,Input,Output

const epsilon = time.Millisecond

func checkTimeStamp(expected, response time.Time) bool {
	return response.After(expected.Add(-epsilon)) && response.Before(expected.Add(epsilon))
}

func getLineWoTime(line string) string {
	return strings.Join(strings.Split(line, " ")[0:2], " ")
}

type TestStimuliResponse []struct {
	Topic             string
	Payload           string
	ExpectedTimeStamp time.Time
	ExpectedLines     []string
}

func testStimuliResponse(
	t *testing.T,
	mockCtrl *gomock.Controller,
	config Config,
	dut HandleFunc,
	stimuli TestStimuliResponse,
) {
	for _, s := range stimuli {
		t.Logf("stimuli: Topic='%s'", s.Topic)
		t.Logf("stimuli: Payload='%s'", s.Payload)
		t.Logf("stimuli: ExpectedLines='%s'", s.ExpectedLines)

		mockInput := converter_mock.NewMockInput(mockCtrl)
		mockInput.EXPECT().Topic().Return(s.Topic).MinTimes(1)
		mockInput.EXPECT().Payload().Return([]byte(s.Payload)).AnyTimes() // must no be called when topic is invalid

		outputTestFuncCounter := 0
		responseLines := make([]string, 0, len(s.ExpectedLines))
		outputTestFunc := func(output Output) {
			outputTestFuncCounter += 1

			point, err := influxDbClient.ToInfluxPoint(output)
			if err != nil {
				t.Errorf("expect no error, got: %v", err)
			}

			response := getLineWoTime(point.String())
			t.Logf("response: '%s'", response)

			responseLines = append(responseLines, response)

			if !checkTimeStamp(s.ExpectedTimeStamp, output.Time()) {
				t.Errorf("expect timestamp to %s but got %s", s.ExpectedTimeStamp, output.Time())
			}
		}
		dut(config, mockInput, outputTestFunc)

		// sort strings before comparison
		sort.Strings(s.ExpectedLines)
		sort.Strings(responseLines)
		if !reflect.DeepEqual(s.ExpectedLines, responseLines) {
			t.Errorf("expected lines do not match response lines:")

			for _, l := range s.ExpectedLines {
				t.Errorf("  expected: %s", l)
			}
			for _, l := range responseLines {
				t.Errorf("  got: %s", l)
			}
		}
	}
}