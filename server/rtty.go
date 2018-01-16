package main

import (
    "flag"
    "log"
    "fmt"
    "time"
    "strconv"
    "math/rand"
    "net/http"
    "crypto/md5"
    "encoding/hex"
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

var devices = make(map[string]*Device)

var upgrader = websocket.Upgrader{}

func generateSID(did string) string {
    md5Ctx := md5.New()
    md5Ctx.Write([]byte(did + strconv.FormatFloat(rand.Float64(), 'e', 6, 32)))
    cipherStr := md5Ctx.Sum(nil)
    fmt.Print(cipherStr)
    fmt.Print("\n")
    return hex.EncodeToString(cipherStr)
}

func aliveDevice(did string) {
    dev := devices[did]
    dev.active = 3
}

func parseFrame(msg []byte) *Frame {
    f := Frame{}
    json.Unmarshal(msg, &f)
    return &f
}

func flushDevice() {
    for {
        time.Sleep(time.Second * 5)

        for did, dev := range devices {
            dev.active--
            if dev.active == 0 {
                delete(devices, did)
            }
        }
    }
}

func login(did string, ws *websocket.Conn) (string, bool) {
    f := Frame{Type: "login"}

    dev, ok := devices[did]
    if !ok {
        f.Err = "Device off-line"
        js, _ := json.Marshal(f)
        ws.WriteMessage(websocket.TextMessage, js)
        return "", false
    }

    sid := generateSID(did)
    dev.sid[sid] = ws

    f.SID = sid
    js, _ := json.Marshal(f)
    ws.WriteMessage(websocket.TextMessage, js)
    dev.ws.WriteMessage(websocket.TextMessage, js)
    return sid, true
}

func logout(did string, sid string) {
    f := Frame{Type: "logout", SID: sid}
    js, _ := json.Marshal(f)
    devices[did].ws.WriteMessage(websocket.TextMessage, js)
}

func wsHandlerDevice(w http.ResponseWriter, r *http.Request) {
    did := r.URL.Query().Get("did")

    c, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Print("upgrade:", err)
        return
    }
    defer c.Close()

    dev := Device{3, c, make(map[string]*websocket.Conn)}
    devices[did] = &dev

    for {
        mt, message, err := c.ReadMessage()
        if err != nil {
            log.Println("read", err)
            break
        }

        if mt == websocket.TextMessage {
            f := parseFrame(message)
            
            if f.Type == "ping" {
                aliveDevice(did)
            } else if (f.Type == "data" || f.Type == "logout") {
                devices[did].sid[f.SID].WriteMessage(websocket.TextMessage, message)
            }
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
    
    sid, ok := login(did, c)
    if !ok {
        return
    }

    defer logout(did, sid)

    for {
        mt, message, err := c.ReadMessage()
        if err != nil {
            log.Println("read:", err)
            break
        }

        if mt == websocket.TextMessage {
            f := parseFrame(message)
            if f.Type == "data" {
                devices[did].ws.WriteMessage(websocket.TextMessage, message)
            }
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
    document := flag.String("document", "./www", "http service document dir")
    log.SetFlags(0)
    flag.Parse()

    go flushDevice()

    http.HandleFunc("/ws/device", wsHandlerDevice)
    http.HandleFunc("/ws/browser", wsHandlerBrowser)
    http.HandleFunc("/list", HandlerList)
    http.Handle("/", http.FileServer(http.Dir(*document)))
    log.Fatal(http.ListenAndServe(":" + strconv.Itoa(*port), nil))
}