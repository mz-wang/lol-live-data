package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var logger *zap.Logger

type TokenResponse struct {
	Token string `json:"token"`
}

func init() {
	l, err := zap.NewDevelopment()
	if err != nil {
		log.SetFlags(0)
		log.Fatal("init: ", err)
	}
	logger = l
}

func issueToken() (*TokenResponse, error) {
	r, err := http.Get("https://api.lolesports.com/api/issueToken")
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	var t TokenResponse
	err = json.Unmarshal(body, &t)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

func main() {
	t, err := issueToken()
	if err != nil {
		logger.Fatal("unable to issue websocket token", zap.Error(err))
	}

	q := fmt.Sprintf("jwt=%s", t.Token)
	u := url.URL{Scheme: "wss", Host: "livestats.proxy.lolesports.com", Path: "/stats", RawQuery: q}

	c, res, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		logger.Fatal("failed to connect to websocket",
			zap.String("url", u.String()),
			zap.String("status", res.Status),
			zap.Error(err),
		)
	}
	defer c.Close()

	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
				logger.Fatal("websocket closed unexpectedly", zap.Error(err))
			}
		}

		var m map[string]interface{}
		err = json.Unmarshal(msg, &m)
		if err != nil {
			logger.Error("failed to unmarshall websocket message", zap.Error(err))
		}

		for k, v := range m {
			if v == nil {
				delete(m, k)
			}
		}

		logger.Info("processed websocket message", zap.Int("count", len(m)))
	}
}
