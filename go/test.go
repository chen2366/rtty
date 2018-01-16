package main

import (
    "flag"
    "log"
    "fmt"
    "strconv"
    "net/http"
    "encoding/json"
    "github.com/gorilla/websocket"
)

type Frame struct {
    Type string `json:"type"`
    SID string `json:"sid"`
    Data string `json:"data"`
    Err string `json:"err"`
}

type Device struct {
    active int
    ws *websocket.Conn
    sid map[string]*websocket.Conn
}

var devices = make(map[string]Device)

var upgrader = websocket.Upgrader{} // use default options

func aliveDevice(did string) {
    dev := devices[did]
    dev.active = 3
}

func parseFrame(msg []byte) *Frame {
    f := Frame{}
    json.Unmarshal(msg, &f)
    return &f
}

func login(did string) string{
    sid := "1231232"
    dev := devices[did]

    f := Frame{Type: "login", SID: sid}
    js, _ := json.Marshal(f)
    dev.ws.WriteMessage(websocket.TextMessage, js)
    return sid
}

func wsHandlerDevice(w http.ResponseWriter, r *http.Request) {
    did := r.URL.Query().Get("did")

    c, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Print("upgrade:", err)
        return
    }
    defer c.Close()
    
    devices[did] = Device{3, c, make(map[string]*websocket.Conn)}

    fmt.Println("new device:", did)

    for {
        mt, message, err := c.ReadMessage()
        if err != nil {
            log.Println("read:", err)
            break
        }

        if mt == websocket.TextMessage {
            f := parseFrame(message)
            
            if f.Type == "ping" {
                aliveDevice(did)
            } else if (f.Type == "data" || f.Type == "logout") {

            }
            fmt.Println("frame type:", f.Type)
        }
    }
}

func wsHandlerBrowser(w http.ResponseWriter, r *http.Request) {
    did := r.URL.Query().Get("did")

    c, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Print("upgrade:", err)
        return
    }
    defer c.Close()

    sid := login(did)

    fmt.Println("new login:", sid)

    for {
        mt, message, err := c.ReadMessage()
        if err != nil {
            log.Println("read:", err)
            break
        }

        if mt == websocket.TextMessage {
            f := parseFrame(message)
            
            if f.Type == "ping" {
                aliveDevice(did)
            } else if (f.Type == "data" || f.Type == "logout") {

            }
            fmt.Println("frame type:", f.Type)
        }
    }
}

func HandlerList(w http.ResponseWriter, r *http.Request) {
    var s []string
    for k, _ := range devices {
        s = append(s, k)
    }
    js, err := json.Marshal(s)
    if err != nil {
        fmt.Println("error:", err)
    } else {
        fmt.Fprintf(w, "%s", js)
    }
}

func main() {
    port := flag.Int("port", 5912, "http service port")
    document := flag.String("document", ".", "http service document dir")
    log.SetFlags(0)
    flag.Parse()
    http.HandleFunc("/ws/device", wsHandlerDevice)
    http.HandleFunc("/ws/browser", wsHandlerBrowser)
    http.HandleFunc("/list", HandlerList)
    http.Handle("/", http.FileServer(http.Dir(*document)))
    log.Fatal(http.ListenAndServe(":" + strconv.Itoa(*port), nil))
}