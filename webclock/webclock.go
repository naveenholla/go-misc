package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"code.google.com/p/go.net/websocket"
)

var logger = log.New(os.Stderr, "webclock ", log.LstdFlags|log.Lshortfile)

var addr string

func init() {
	http.HandleFunc("/", index)
	http.Handle("/ws", websocket.Handler(handle))
}

const indexHtml = `
<html>
<head>
  <title>Web clock</title>
  <script type="text/javascript" src="http://ajax.googleapis.com/ajax/libs/jquery/1.4.2/jquery.min.js"></script>
  <script type="text/javascript">
  	$(function() {
		var conn;
		var WS = window["WebSocket"] ? WebSocket : MozWebSocket;

		function onMessage(tick) {
			$("#ticks").append($("<li/>").text(tick.data));
		};

		// Send callback.
		$("#form").submit(function() {
			if (conn) conn.close();
			try {
				$("#ticks").empty();
				conn = new WS("ws://%s/ws");
				conn.onopen = function() {
					// connection opening is asynchronous.
					var req = $("#freq").val();
					conn.send(req);
				};
				conn.onmessage = onMessage;
			} catch(ex) {
				$("#ticks").append($("<li/>").text(ex.toString()));
			}
			return false;
		});
	});
  </script>
</head>
<body>
  <h1>Web clock</h1>

  <form id="form">
	<input type="submit" value="Set frequency:" />
	<input type="text" id="freq" size="64" value="1s"/>
  </form>

  <p>Time is ...</p>
  <ul id="ticks">
  </ul>
</body>
</html>
`

func index(resp http.ResponseWriter, req *http.Request) {
	logger.Printf("new http client: %s", req.RemoteAddr)
	fmt.Fprintf(resp, indexHtml, addr)
}

func handle(conn *websocket.Conn) {
	logger.Printf("new websocket client: %s", conn.RemoteAddr())
	defer conn.Close()
	req := "1s"
	if err := websocket.Message.Receive(conn, &req); err != nil {
		logger.Printf("error request: %s", err)
		return
	}
	if freq, err := time.ParseDuration(req); err != nil {
		logger.Printf("invalid request: %s", req)
		websocket.Message.Send(conn, "invalid request")
		return
	} else {
		logger.Printf("new request for ticks from %s, period=%s", conn.RemoteAddr(), freq)
		T := time.NewTicker(freq)
		defer T.Stop()
		ticks := 0
		for t := range T.C {
			if ticks >= 16 {
				return
			}
			err := websocket.Message.Send(conn, t.String())
			if err != nil {
				logger.Printf("error sending: %s", err)
				return
			}
			ticks++
		}
	}
}

func main() {
	flag.StringVar(&addr, "http", "localhost:8082", "listen here")
	flag.Parse()

	logger.Printf("serving on %s", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		logger.Printf("error in server: %s", err)
	}
}
