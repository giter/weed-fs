package main

import (
  "libs/storage"
  "flag"
  "fmt"
  "net/http"
  "encoding/json"
  "log"
  "mime"
  "math/rand"
  "strconv"
  "strings"
  "time"
)

var (

  port        = flag.Int("port", 8080, "http listen port")
  chunkFolder = flag.String("dir", "/tmp", "data directory to store files")
  volumes     = flag.String("volumes", "0,1-3,4", "comma-separated list of volume ids or range of ids")
  publicUrl   = flag.String("publicUrl", "localhost:8080", "public url to serve data read")
  metaServer  = flag.String("mserver", "localhost:9333", "master directory server to store mappings")
  IsDebug     = flag.Bool("debug", false, "enable debug mode")
  pulse       = flag.Int("pulseSeconds", 5, "number of seconds between heartbeats")

  store *storage.Store
)

func statusHandler(w http.ResponseWriter, r *http.Request) {
  writeJson(w, r, store.Status())
}

func addVolumeHandler(w http.ResponseWriter, r *http.Request) {
  store.AddVolume(r.FormValue("volume"))
  writeJson(w, r, store.Status())
}

func storeHandler(w http.ResponseWriter, r *http.Request) {

  switch r.Method {
  case "GET":
    GetHandler(w, r)
  case "DELETE":
    DeleteHandler(w, r)
  case "POST":
    PostHandler(w, r)
  }
}

func GetHandler(w http.ResponseWriter, r *http.Request) {

  n := new(storage.Needle)

  vid, fid, ext := parseURLPath(r.URL.Path)
  volumeId, _ := strconv.ParseUint(vid,10,64)
  n.ParsePath(fid)

  if *IsDebug {
    log.Println("volume", volumeId, "reading", n)
  }

  cookie := n.Cookie
  count, e := store.Read(volumeId, n)

  if *IsDebug {
    log.Println("read bytes", count, "error", e)
  }

  if n.Cookie != cookie {
    log.Println("request with unmaching cookie from ", r.RemoteAddr, "agent", r.UserAgent())
    return
  }

  if ext != "" {
    w.Header().Set("Content-Type", mime.TypeByExtension(ext))
  }

  w.Write(n.Data)
}

func PostHandler(w http.ResponseWriter, r *http.Request) {

  vid, _, _ := parseURLPath(r.URL.Path)
  volumeId, e := strconv.ParseUint(vid,10,64)

  if e != nil {
    writeJson(w, r, e)
  } else {
    ret := store.Write(volumeId, storage.NewNeedle(r))
    m := make(map[string]uint32)
    m["size"] = ret
    writeJson(w, r, m)
  }
}

func DeleteHandler(w http.ResponseWriter, r *http.Request) {

  n := new(storage.Needle)
  vid, fid, _ := parseURLPath(r.URL.Path)
  volumeId, _ := strconv.ParseUint(vid,10,64)
  n.ParsePath(fid)

  if *IsDebug {
    log.Println("deleting", n)
  }

  cookie := n.Cookie
  count, ok := store.Read(volumeId, n)

  if ok!=nil {
    m := make(map[string]uint32)
    m["size"] = 0
    writeJson(w, r, m)
    return
  }

  if n.Cookie != cookie {
    log.Println("delete with unmaching cookie from ", r.RemoteAddr, "agent", r.UserAgent())
    return
  }

  n.Size = 0
  store.Delete(volumeId, n)
  m := make(map[string]uint32)
  m["size"] = uint32(count)
  writeJson(w, r, m)
}

func writeJson(w http.ResponseWriter, r *http.Request, obj interface{}) {

  w.Header().Set("Content-Type", "application/javascript")
  bytes, _ := json.Marshal(obj)
  callback := r.FormValue("callback")

  if callback == "" {
    w.Write(bytes)
  } else {
    w.Write([]uint8(callback))
    w.Write([]uint8("("))
    fmt.Fprint(w, string(bytes))
    w.Write([]uint8(")"))
  }
}

func parseURLPath(path string) (vid, fid, ext string) {

  sepIndex := strings.LastIndex(path, "/")
  commaIndex := strings.LastIndex(path[sepIndex:], ",")

  if commaIndex <= 0 {
    log.Println("unknown file id", path[sepIndex+1:])
    return
  }

  dotIndex := strings.LastIndex(path[sepIndex:], ".")
  vid = path[sepIndex+1 : commaIndex]
  fid = path[commaIndex+1:]
  ext = ""

  if dotIndex > 0 {
    fid = path[commaIndex+1 : dotIndex]
    ext = path[dotIndex+1:]
  }

  return 
}

func main() {

  flag.Parse()

  store = storage.NewStore(*port, *publicUrl, *chunkFolder, *volumes)
  defer store.Close()

  http.HandleFunc("/", storeHandler)
  http.HandleFunc("/status", statusHandler)
  http.HandleFunc("/add_volume", addVolumeHandler)

  go func() {
    for {
      store.Join(*metaServer)
      ns := int64(*pulse) * 1e9
      sl := time.Duration(ns + rand.Int63()%ns)
      time.Sleep(sl)
    }
  }()

  log.Println("store joined at", *metaServer)

  log.Println("Start storage service at http://127.0.0.1:"+strconv.Itoa(*port), "public url", *publicUrl)
  
  srv := &http.Server{
                Addr:":"+strconv.Itoa(*port),
                Handler: http.DefaultServeMux,
                ReadTimeout: 30*time.Second,
        }

    e := srv.ListenAndServe()

  if e != nil {
    log.Fatalf("Fail to start:", e.Error())
  }

}
