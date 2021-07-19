package controllers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/runatlantis/atlantis/server/controllers"
	"github.com/runatlantis/atlantis/server/events/models"
	"github.com/runatlantis/atlantis/server/logging"

	. "github.com/petergtz/pegomock"
	"github.com/runatlantis/atlantis/server/controllers/mocks"
	"github.com/runatlantis/atlantis/server/controllers/mocks/matchers"
	//. "github.com/runatlantis/atlantis/testing"
)

func TestGetLogStream_WebSockets(t *testing.T) {
	t.Run("Should Group by Project Info", func(t *testing.T) {
		RegisterMockTestingT(t)
		tempchan := make(chan *models.TerraformOutputLine)
		websocketMock := mocks.NewMockWebsocketHandler()
		websocketWriterMock := mocks.NewMockWebsocketResponseWriter()
		params := map[string]string{
			"org":     "test-org",
			"repo":    "test-repo",
			"pull":    "1",
			"project": "test-project",
		}
		ctx := context.Background()
		ctx = context.WithValue(ctx, int(0), params)
		request, _ := http.NewRequest(http.MethodGet, "/logStreaming/org/repo/1/project/ws", nil)
		request = mux.SetURLVars(request, params)
		response := httptest.NewRecorder()
		logStreamingController := &controllers.LogStreamingController{
			Logger:              logging.NewNoopLogger(t),
			TerraformOutputChan: tempchan,
			WebsocketHandler:    websocketMock,
		}
		When(websocketMock.Upgrade(matchers.AnyHttpResponseWriter(), matchers.AnyPtrToHttpRequest(), matchers.AnyHttpHeader())).ThenReturn(websocketWriterMock, nil)
		go func() {
			tempchan <- &models.TerraformOutputLine{
				ProjectInfo: "test-org/test-repo/1/test-project",
				Line:        "Test Terraform Output",
			}
		}()
		go func() {
			logStreamingController.Listen()
		}()

		go func() {
			logStreamingController.GetLogStreamWS(response, request)
		}()
		time.Sleep(1 * time.Second)
		websocketWriterMock.VerifyWasCalled(Once()).WriteMessage(AnyInt(), matchers.EqSliceOfByte([]byte("Test Terraform Output")))
	})
} //make sure write message is not being called when diff project info
//add test to plan command runner
//nested structs with gorilla and websocker that implements upgrade and implements interface
