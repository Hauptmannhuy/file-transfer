package signaling

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"golang.org/x/net/websocket"
)

type RegisterMessage struct {
	IP      int    `json:"ip"`
	Port    int    `json:"port"`
	CodeMsg string `json:"code"`
}

var registerPeers []RegisterMessage = []RegisterMessage{}

type signalServer struct {
	wsConnections map[string]*websocket.Conn
}

func Listen() {
	server := signalServer{
		wsConnections: map[string]*websocket.Conn{},
	}

	http.Handle("POST /register", websocket.Server{
		Handler: func(c *websocket.Conn) {
			server.handleWS(c)
		},
	})

	http.HandleFunc("POST /register", func(w http.ResponseWriter, r *http.Request) {
		var buffer []byte
		var regMessage RegisterMessage

		_, err := r.Body.Read(buffer)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			r.Body.Close()
		}

		if err := json.Unmarshal(buffer, &regMessage); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			r.Body.Close()
		}

		registerPeers = append(registerPeers, regMessage)

	})

	http.HandleFunc("POST /get_user", func(w http.ResponseWriter, r *http.Request) {
		var buffer []byte
		var getUserMsg RegisterMessage

		_, err := r.Body.Read(buffer)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			r.Body.Close()
		}

		if err := json.Unmarshal(buffer, &getUserMsg); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			r.Body.Close()
		}

		for _, msg := range registerPeers {
			if msg.CodeMsg == getUserMsg.CodeMsg && msg.IP != getUserMsg.IP && msg.Port != getUserMsg.Port {
				data, err := json.Marshal(msg)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
				}
				if _, err := w.Write(data); err != nil {
					return
				}

			}
		}
	})
	slog.Info("here00")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		slog.Info("error signal server", "err", err)
	}
	slog.Info("here11")

}

func makeRegisterKey(msg RegisterMessage) string {
	return strconv.Itoa(msg.IP) + strconv.Itoa(msg.Port) + msg.CodeMsg
}

func (server *signalServer) handleWS(conn *websocket.Conn) {
	header := conn.Config().Header
	id := header.Get("id")
	server.wsConnections[id] = conn

	go func() {
		var buf []byte
		_, err := conn.Read(buf)
		if err != nil {
			slog.Info("err reading client ws conn", "err", err)
		}
	}()
}
